package scheduling

import (
	"errors"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/logging"
	"log"
	"math"
	"sync/atomic"
	"time"
)

var dropCountPtr = new(int64)
var expirationInterval = time.Duration(config.GetInt("policy.drop.expiration", 30))

type GreedyPolicy struct {
	dropManager *DropManager
}

type DropManager struct {
	dropChan   chan time.Time
	dropCount  int64
	expiration int64
}

func (p *GreedyPolicy) Init() {
	p.dropManager = &DropManager{
		dropCount:  0,
		dropChan:   make(chan time.Time, 1),
		expiration: time.Now().UnixNano(),
	}

	go p.dropManager.dropRun()
}

func (p *GreedyPolicy) OnCompletion(r *scheduledRequest) {
	return
}

func (p *GreedyPolicy) OnArrival(r *scheduledRequest) {
	offloading := config.GetBool("offloading", false)
	containerID, err := acquireWarmContainer(r.Fun)
	if err == nil {
		log.Printf("Using a warm container for: %v", r)
		execLocally(r, containerID, true)
	} else if errors.Is(err, NoWarmFoundErr) && offloading {
		act := takeSchedulingDecision(r)
		switch act {
		case SCHED_BASIC:
			handleColdStart(r, true)
			break
		case SCHED_LOCAL:
			handleColdStart(r, false)
			break
		case SCHED_REMOTE:
			handleOffload(r)
			break
		case SCHED_DROP:
			p.dropManager.sendDropAlert()
			dropRequest(r)
			break
		default:
			p.dropManager.sendDropAlert()
			dropRequest(r)
		}
	} else {
		//offloading disabled
		handleColdStart(r, false)
	}
}

func takeSchedulingDecision(r *scheduledRequest) (act schedulingDecision) {
	var timeLocal, timeOffload float64
	logger := logging.GetLogger()
	localStatus, _ := logger.GetLocalLogStatus(r.Fun.Name)
	remoteStatus, _ := logger.GetRemoteLogStatus(r.Fun.Name)
	if math.IsNaN(localStatus.AvgExecutionTime) || math.IsNaN(remoteStatus.AvgExecutionTime) {
		//not enough information, use basic schedule
		return SCHED_BASIC
	}

	timeLocal = localStatus.AvgColdInitTime + localStatus.AvgExecutionTime
	timeOffload = (remoteStatus.AvgColdInitTime+remoteStatus.AvgWarmInitTime)/float64(2) + remoteStatus.AvgExecutionTime + remoteStatus.AvgOffloadingLatency
	node.RLock()
	defer node.RUnlock()
	if node.AvailableMemMB < r.Fun.MemoryMB { //not enough memory
		if r.RequestQoS.MaxRespT <= timeOffload {
			return SCHED_REMOTE
		} else { //not enough memory and offloading takes too long
			return SCHED_DROP
		}
	}
	return decision(timeOffload, timeLocal, r)
}

func decision(timeOffload float64, timeLocal float64, r *scheduledRequest) (act schedulingDecision) {
	if timeLocal > r.RequestQoS.MaxRespT && timeOffload <= r.RequestQoS.MaxRespT {
		return SCHED_REMOTE
	} else if timeLocal <= r.RequestQoS.MaxRespT && timeOffload > r.RequestQoS.MaxRespT {
		return SCHED_LOCAL
	} else if timeLocal > r.RequestQoS.MaxRespT && timeOffload > r.RequestQoS.MaxRespT {
		return SCHED_DROP
	}

	// timeLocal  <= r.QoS.MaxRespT && timeOffload <= r.QoS.MaxRespT
	switch r.Class {
	case function.LOW:
		// a request has been dropped recently -> do offload in a conservative fashion
		if *dropCountPtr > 0 {
			return SCHED_REMOTE
		} else {
			return SCHED_LOCAL
		}
	case function.HIGH_PERFORMANCE:
		return SCHED_LOCAL
	case function.HIGH_AVAILABILITY:
		return SCHED_REMOTE
	}

	//never used here
	return SCHED_BASIC
}

func (d *DropManager) sendDropAlert() {
	dropTime := time.Now()
	if dropTime.UnixNano() > d.expiration {
		select { //non-blocking write on channel
		case d.dropChan <- dropTime:
			return
		default:
			return
		}
	}
}

func (d *DropManager) dropRun() {
	ticker := time.NewTicker(time.Duration(config.GetInt("policy.drop.expiration", 30)) * time.Second)
	for {
		select {
		case tick := <-d.dropChan:
			log.Printf("drop occurred")
			//update expiration
			d.expiration = tick.Add(expirationInterval * time.Second).UnixNano()
			atomic.AddInt64(&d.dropCount, 1)
		case <-ticker.C:
			if time.Now().UnixNano() >= d.expiration {
				log.Printf("drop expiration timer exceded")
				atomic.StoreInt64(&d.dropCount, 0)
			}
		}
	}
}
