package scheduling

import (
	"github.com/grussorusso/serverledge/internal/node"
	"log"
)

// CloudEdgePolicy supports only Edge-Cloud Offloading
type CloudEdgePolicy struct{}

func (p *CloudEdgePolicy) Init() {
	InitDropManager()
}

func (p *CloudEdgePolicy) OnCompletion(r *scheduledRequest) {

}

func (p *CloudEdgePolicy) OnArrival(r *scheduledRequest) {
	containerID, err := node.AcquireWarmContainer(r.Fun)
	if err == nil {
		log.Printf("Using a warm container for: %v", r)
		execLocally(r, containerID, true)
	} else if handleColdStart(r) {
		return
	} else {
		handleCloudOffload(r)
	}
}
