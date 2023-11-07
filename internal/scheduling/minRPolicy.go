package scheduling

import (
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/node"
	"log"
	"math/rand"
	"time"
)

// MinRPolicy: greedy policy that chooses scheduling action based on the minimum expected latency
type MinRPolicy struct {
}

var grabber metricGrabber

var latencyCloud = 0.0
var latencyLocal = 0.0

func queryDb() {
	evaluationTicker :=
		time.NewTicker(evaluationInterval)

	for {
		select {
		case _ = <-evaluationTicker.C: // Evaluation handler
			s := rand.NewSource(time.Now().UnixNano())
			rGen = rand.New(s)
			log.Println("Query Db")

			// Query DB to get metrics
			//d.deleteOldData(24 * time.Hour)
			grabber.GrabMetrics()
		}
	}
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
	go queryDb()
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

	if canExecute(r.Fun) && (latencyLocal < latencyCloud) {
		// Execute locally
		containerID, err := node.AcquireWarmContainer(r.Fun)
		if err == nil {
			execLocally(r, containerID, true)
		} else if handleColdStart(r) {
			return
		} else {
			dropRequest(r)
		}
	} else if r.CanDoOffloading {
		// Execute offloading to cloud
		handleCloudOffload(r)
	} else {
		// Drop request
		dropRequest(r)
	}
}
