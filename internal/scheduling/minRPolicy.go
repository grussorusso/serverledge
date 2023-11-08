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
	log.Println("Latency local: ", latencyLocal)
	log.Println("Latency cloud: ", latencyCloud)
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
	log.Println("INITIALIZING GRABBER")
	grabber.InitMetricGrabber()
	log.Println("GRABBER INITIALIZED")
}

func (p *MinRPolicy) OnCompletion(r *scheduledRequest) {
	log.Println("COMPLETED ACTION")
	if r.ExecReport.SchedAction == SCHED_ACTION_OFFLOAD_CLOUD {
		log.Println("SENDING COMPLETE TO GRABBER")
		grabber.Completed(r, OFFLOADED_CLOUD)
		log.Println("COMPLETE SENT")
	} else if r.ExecReport.SchedAction == SCHED_ACTION_OFFLOAD_EDGE {
		log.Println("SENDING COMPLETE TO GRABBER")
		grabber.Completed(r, OFFLOADED_EDGE)
		log.Println("COMPLETE SENT")
	} else {
		log.Println("SENDING COMPLETE TO GRABBER")
		grabber.Completed(r, LOCAL)
		log.Println("COMPLETE SENT")
	}
}

func (p *MinRPolicy) OnArrival(r *scheduledRequest) {
	// New arrival
	log.Println("SENDING ARRIVAL")
	class := r.ClassService
	arrivalChannel <- arrivalRequest{r, class.Name}

	// Estimate new latency
	log.Println("ESTIMATING LATENCY")
	latencyLocal, latencyCloud := estimateLatency(r)

	log.Println("DECIDING WHERE TO EXECUTE")
	if latencyLocal < latencyCloud {
		// Execute locally
		log.Println("DECIDED LOCAL")
		containerID, err := node.AcquireWarmContainer(r.Fun)
		if err == nil {
			execLocally(r, containerID, true)
		} else if handleColdStart(r) {
			return
		} else if r.CanDoOffloading {
			log.Println("CAN'T LOCAL GOING CLOUD")
			handleCloudOffload(r)
		} else {
			log.Println("CAN'T CLOUD DROPPING")
			dropRequest(r)
		}
	} else if r.CanDoOffloading {
		log.Println("DECIDED CLOUD")
		// Execute offloading to cloud
		handleCloudOffload(r)
	} else {
		log.Println("DROPPING")
		dropRequest(r)
	}
}
