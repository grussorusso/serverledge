package scheduling

import (
	"log"

	"github.com/grussorusso/serverledge/internal/node"
)

// EdgePolicy supports only Edge-Edge offloading
type OffloadToCluster struct{}

func (p *OffloadToCluster) Init() {
}

func (p *OffloadToCluster) OnCompletion(r *scheduledRequest) {

}

func (p *OffloadToCluster) OnArrival(r *scheduledRequest) {
	containerID, err := node.AcquireWarmContainer(r.Fun)
	if err == nil {
		log.Printf("Using a warm container for: %v", r)
		execLocally(r, containerID, true)
		return
	} else if handleColdStart(r) {
		return
	} else if r.CanDoOffloading {
		handleCloudOffload(r)
		return
	}

	dropRequest(r)
}
