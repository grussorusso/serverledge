package scheduling

import (
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/node"
	"log"
)

// MinRPolicy: greedy policy that chooses scheduling action based on the minimum expected latency
type MinRPolicy struct {
}

var grabber metricGrabber

var latencyCloud = 0.0
var latencyLocal = 0.0

func estimateLatency(r *scheduledRequest) (float64, float64) {
	// Execute a type assertion to access m
	fun, prs := grabber.GrabFunctionInfo(r.Fun.Name)
	if !prs {
		return latencyLocal, latencyCloud
	}
	latencyLocal = fun.meanDuration[0] + fun.probCold[0]*fun.initTime[0]
	latencyCloud = fun.meanDuration[1] + fun.probCold[1]*fun.initTime[1] +
		2*CloudOffloadLatency +
		fun.meanInputSize*8/1000/1000/config.GetFloat(config.BANDWIDTH_CLOUD, 1.0)
	// FIXME AUDIT log.Println("Latency local: ", latencyLocal)
	// FIXME AUDIT log.Println("Latency cloud: ", latencyCloud)
	return latencyLocal, latencyCloud
}

func (p *MinRPolicy) Init() {
	// Initialize DecisionEngine to recover information about incoming requests and metrics
	version := config.GetString(config.STORAGE_VERSION, "flux")
	if version == "mem" {
		// fixme ADD METRIC GRABBER MEM NOT WORKING NOW
	} else {
		grabber = &metricGrabberFlux{}
	}

	log.Println("Scheduler version:", version)
	grabber.InitMetricGrabber()
}

func (p *MinRPolicy) OnCompletion(r *scheduledRequest) {
	if r.ExecReport.SchedAction == SCHED_ACTION_OFFLOAD_CLOUD {
		grabber.Completed(r, OFFLOADED_CLOUD)
	} else if r.ExecReport.SchedAction == SCHED_ACTION_OFFLOAD_EDGE {
		grabber.Completed(r, OFFLOADED_EDGE)
	} else {
		grabber.Completed(r, LOCAL)
	}
}

func (p *MinRPolicy) OnArrival(r *scheduledRequest) {
	// New arrival
	class := r.ClassService
	arrivalChannel <- arrivalRequest{r, class.Name}

	// Estimate new latency
	latencyLocal, latencyCloud := estimateLatency(r)

	if latencyLocal < latencyCloud {
		// Execute locally
		containerID, err := node.AcquireWarmContainer(r.Fun)
		if err == nil {
			execLocally(r, containerID, true)
		} else if handleColdStart(r) {
			return
		} else if r.CanDoOffloading {
			handleCloudOffload(r)
		} else {
			dropRequest(r)
		}
	} else if r.CanDoOffloading {
		// Execute offloading to cloud
		handleCloudOffload(r)
	} else {
		dropRequest(r)
	}
}
