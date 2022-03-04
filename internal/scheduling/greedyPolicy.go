package scheduling

import (
	"errors"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/logging"
	"github.com/grussorusso/serverledge/internal/node"
	"log"
	"math"
)

type GreedyPolicy struct {
}

func (p *GreedyPolicy) Init() {
	InitDropManager()
}

func (p *GreedyPolicy) OnCompletion(r *scheduledRequest) {
	return
}

func (p *GreedyPolicy) OnArrival(r *scheduledRequest) {
	offloading := config.GetBool("offloading", false)
	containerID, err := node.AcquireWarmContainer(r.Fun)
	if err == nil {
		log.Printf("Using a warm container for: %v", r)
		execLocally(r, containerID, true)
	} else if errors.Is(err, node.NoWarmFoundErr) && offloading {
		act := p.takeSchedulingDecision(r)
		switch act {
		case SCHED_BASIC:
			handleColdStart(r) // offload true
			break
		case SCHED_LOCAL:
			handleColdStart(r) // offload false
			break
		case SCHED_REMOTE:
			handleOffload(r, "")
			break
		case SCHED_DROP:
			dropManager.sendDropAlert()
			dropRequest(r)
			break
		default:
			dropManager.sendDropAlert()
			dropRequest(r)
		}
	} else {
		//offloading disabled
		handleColdStart(r)
	}
}

func (p *GreedyPolicy) takeSchedulingDecision(r *scheduledRequest) (act schedulingDecision) {
	var timeLocal, timeOffload float64
	logger := logging.GetLogger()
	localStatus, _ := logger.GetLocalLogStatus(r.Fun.Name)
	remoteStatus, _ := logger.GetRemoteLogStatus(r.Fun.Name)
	if math.IsNaN(localStatus.AvgExecutionTime) || math.IsNaN(remoteStatus.AvgExecutionTime) {
		//not enough information, use basic schedule
		return SCHED_BASIC
	}

	timeLocal = localStatus.AvgColdInitTime + localStatus.AvgExecutionTime
	//todo cold start prediction fix
	timeOffload = (remoteStatus.AvgColdInitTime+remoteStatus.AvgWarmInitTime)/float64(2) + remoteStatus.AvgExecutionTime + remoteStatus.AvgOffloadingLatency
	node.Resources.RLock()
	defer node.Resources.RUnlock()
	if node.Resources.AvailableMemMB < r.Fun.MemoryMB { //not enough memory
		if r.RequestQoS.MaxRespT <= timeOffload {
			return SCHED_REMOTE
		} else { //not enough memory and offloading takes too long
			return SCHED_DROP
		}
	}
	return p.decision(timeOffload, timeLocal, r)
}

func (p *GreedyPolicy) decision(timeOffload float64, timeLocal float64, r *scheduledRequest) (act schedulingDecision) {
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
		if dropManager.dropCount > 0 {
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
