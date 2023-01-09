package scheduling

import (
	"log"

	"github.com/grussorusso/serverledge/internal/node"
)

// EdgePolicy supports only Edge-Edge offloading
type EdgePolicy struct{}

func (p *EdgePolicy) Init() {
}

func (p *EdgePolicy) OnCompletion(r *scheduledRequest) {

}

func (p *EdgePolicy) OnArrival(r *scheduledRequest) {
	if r.CanDoOffloading {
		url := pickEdgeNodeForOffloading(r)
		if url != "" {
			handleEdgeOffload(r, url)
			return
		}
	} else {
		containerID, err := node.AcquireWarmContainer(r.Fun)
		if err == nil {
			log.Printf("Using a warm container for: %v", r)
			execLocally(r, containerID, true)
		} else if handleColdStart(r) {
			return
		}
	}

	dropRequest(r)
}
