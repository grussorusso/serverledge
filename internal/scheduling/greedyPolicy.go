package scheduling

import (
	"errors"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/logging"
	"log"
	"math"
	"sync/atomic"
)

var dropCountPtr = new(int64)

type GreedyPolicy struct{}

func (p *GreedyPolicy) Init() {

}

func (p *GreedyPolicy) OnCompletion(r *scheduledRequest) {

}

func (p *GreedyPolicy) OnArrival(r *scheduledRequest) {
	containerID, err := acquireWarmContainer(r.Fun)
	if err == nil {
		log.Printf("Using a warm container for: %v", r)
		execLocally(r, containerID, true)
	} else if errors.Is(err, NoWarmFoundErr) {
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
			dropRequest(r)
			break
		default:
			dropRequest(r)
		}
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
	node.Lock()
	defer node.Unlock()
	if node.AvailableMemMB < r.Fun.MemoryMB { //not enough memory
		if r.RequestQoS.MaxRespT <= timeOffload {
			return SCHED_REMOTE
		} else { //TODO handle this drop in a different manner
			//not enough memory and offloading takes too long
			atomic.AddInt64(dropCountPtr, 1)
			return SCHED_DROP
		}
	}

	atomic.StoreInt64(dropCountPtr, 0) // memory is now available
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
