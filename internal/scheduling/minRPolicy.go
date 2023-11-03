package scheduling

import (
	"github.com/grussorusso/serverledge/internal/config"
	"log"
)

// MinRPolicy: greedy policy that chooses scheduling action based on the minimum expected latency
type MinRPolicy struct {
}

var mg metricGrabber

func estimateLatency() (float64, float64) {
	var latencyCloud = 0.0
	var latencyLocal = 0.0
	return latencyLocal, latencyCloud
}
func (p *MinRPolicy) Init() {
	// Initialize DecisionEngine to recover information about incoming requests and metrics
	version := config.GetString(config.STORAGE_VERSION, "flux")
	if version == "mem" {
		// fixme ADD METRIC GRABBER MEM NOT WORKING NOW
	} else {
		mg = &metricGrabberFlux{}
	}

	log.Println("Scheduler version:", version)
	mg.InitMetricGrabber()
}

func (p *MinRPolicy) OnCompletion(r *scheduledRequest) {

}

func (p *MinRPolicy) OnArrival(r *scheduledRequest) {

}
