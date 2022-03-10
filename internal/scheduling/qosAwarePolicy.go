package scheduling

import (
	"log"
	"math"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/logging"
	"github.com/grussorusso/serverledge/internal/node"
	"github.com/grussorusso/serverledge/internal/registration"
)

var remoteServerUrl = config.GetString(config.CLOUD_URL, "http://127.0.0.1:1323")

type QosAwarePolicy struct{}

func (p *QosAwarePolicy) Init() {
	InitDropManager()
}

func (p *QosAwarePolicy) OnCompletion(r *scheduledRequest) {

}

func (p *QosAwarePolicy) OnArrival(r *scheduledRequest) {
	//offloading := config.GetBool("offloading", false)
	containerID, err := node.AcquireWarmContainer(r.Fun)
	if err == nil {
		log.Printf("Using a warm container for: %v", r)
		execLocally(r, containerID, true)
	} else {
		if r.CanDoOffloading {
			p.takeSchedulingDecision(r)
		} else {
			if !handleColdStart(r) {
				dropRequest(r)
			}
		}

	}
}

func (p *QosAwarePolicy) takeSchedulingDecision(r *scheduledRequest) {
	switch r.RequestQoS.Class {
	case function.LOW:
		handleLowReq(r)
	case function.HIGH_PERFORMANCE:
		handleHighPerfReq(r)
	case function.HIGH_AVAILABILITY:
		handleHAReq(r)
	default:
		dropRequest(r)
	}

}

//handleHAReq handler for HIGH_AVAILABILITY service class, idea: if it is possible do offload to the cloud
//else process in local. Drop the request if there aren't enough resources.
//Here Edge-offloading is not possible because it is not safe, we are not certain that the target edge-server is up & running
func handleHAReq(r *scheduledRequest) {
	var timeLocal, timeOffload float64
	logger := logging.GetLogger()
	localStatus, _ := logger.GetLocalLogStatus(r.Fun.Name)
	remoteStatus, _ := logger.GetRemoteLogStatus(r.Fun.Name)
	if math.IsNaN(localStatus.AvgExecutionTime) || math.IsNaN(remoteStatus.AvgWarmInitTime) ||
		math.IsNaN(remoteStatus.AvgColdInitTime) || math.IsNaN(remoteStatus.AvgExecutionTime) || math.IsNaN(remoteStatus.AvgOffloadingLatency) {
		//not enough information, remote (cloud schedule)
		handleOffload(r, remoteServerUrl)
		return
	}

	timeLocal = localStatus.AvgColdInitTime + localStatus.AvgExecutionTime
	timeOffload = (remoteStatus.AvgColdInitTime+remoteStatus.AvgWarmInitTime)/float64(2) + remoteStatus.AvgExecutionTime + remoteStatus.AvgOffloadingLatency
	if timeOffload <= r.RequestQoS.MaxRespT { // todo add error-gap
		handleOffload(r, remoteServerUrl)
		return
	}
	//(cloud) offload takes too long
	if timeLocal <= r.RequestQoS.MaxRespT {
		if handleColdStart(r) {
			return
		}
	}

	//timeLocal > r.RequestQoS.MaxRespT && timeOffload > r.RequestQoS.MaxRespT
	dropRequest(r)
}

//handleHighPerfReq handler for HIGH_PERFORMANCE service class; idea: if it is possible process the request in local,
//else if QoS maxResponse time is not exceeded do offload.
//Before drop try to offload to an appropriate edge server, if any.
func handleHighPerfReq(r *scheduledRequest) {
	var timeLocal, timeOffload float64
	logger := logging.GetLogger()
	localStatus, _ := logger.GetLocalLogStatus(r.Fun.Name)
	remoteStatus, _ := logger.GetRemoteLogStatus(r.Fun.Name)
	if math.IsNaN(localStatus.AvgExecutionTime) || math.IsNaN(remoteStatus.AvgWarmInitTime) ||
		math.IsNaN(remoteStatus.AvgColdInitTime) || math.IsNaN(remoteStatus.AvgExecutionTime) || math.IsNaN(remoteStatus.AvgOffloadingLatency) {
		//not enough information, local schedule with cold start
		if !handleColdStart(r) {
			dropRequest(r)
		}
		return
	}

	timeLocal = localStatus.AvgColdInitTime + localStatus.AvgExecutionTime
	timeOffload = (remoteStatus.AvgColdInitTime+remoteStatus.AvgWarmInitTime)/float64(2) + remoteStatus.AvgExecutionTime + remoteStatus.AvgOffloadingLatency
	if timeLocal <= r.RequestQoS.MaxRespT {
		if handleColdStart(r) {
			return
		}
	}
	//cold start takes too long, or it is not possible (resources unavailable)
	if timeOffload <= r.RequestQoS.MaxRespT {
		handleOffload(r, remoteServerUrl)
		return
	}

	//timeLocal > r.RequestQoS.MaxRespT && timeOffload > r.RequestQoS.MaxRespT
	if registration.Reg.RwMtx.TryLock() {
		defer registration.Reg.RwMtx.Unlock()
		url := handleEdgeOffloading(r)
		if url != "" {
			handleOffload(r, url)
			return
		} else {
			dropRequest(r)
		}
	}
	dropRequest(r)
}

//handleLowReq handler for LOW service class; idea: best-effort service.
//If cold start is not possible try to do edge-offloading.
//Before drop do offload to the cloud, cloud resources are more expensive, not waste them for low-class requests
func handleLowReq(r *scheduledRequest) {
	logger := logging.GetLogger()
	remoteStatus, _ := logger.GetRemoteLogStatus(r.Fun.Name)
	if math.IsNaN(remoteStatus.AvgExecutionTime) {
		//not enough remote information, do (cloud) offload opportunistically
		handleOffload(r, remoteServerUrl)
		return
	}

	if handleColdStart(r) {
		return
	}
	//not enough resources, try edge-offloading
	if registration.Reg.RwMtx.TryLock() {
		defer registration.Reg.RwMtx.Unlock()
		url := handleEdgeOffloading(r)
		if url != "" {
			handleOffload(r, url)
			return
		}
	}
	//edge offload not possible
	handleOffload(r, remoteServerUrl)
}

func handleEdgeOffloading(r *scheduledRequest) (url string) {
	nearbyServersMap := registration.Reg.NearbyServersMap
	newMap := make(map[string]*registration.StatusInformation)
	for k, v := range nearbyServersMap {
		if v.DropCount == 0 {
			newMap[k] = v
		}
	}
	for _, v := range newMap {
		if v.AvailableMemMB >= r.Request.Fun.MemoryMB && v.AvailableCPUs >= r.Request.Fun.CPUDemand {
			return v.Url
		}
	}

	return ""
}
