package scheduling

import (
	"log"

	"github.com/grussorusso/serverledge/internal/node"
)

// EdgePolicy supports only Edge-Edge offloading. Always does offloading to an edge node if enabled. When offloading is not enabled executes the request locally.
type EdgePolicy struct{}

func (p *EdgePolicy) Init() {
}

func (p *EdgePolicy) OnCompletion(r *scheduledRequest) {

}

func (p *EdgePolicy) OnArrival(r *scheduledRequest) {
	if r.CanDoOffloading {
		url := pickEdgeNodeForOffloading(r)
		if url != "" {
			handleOffload(r, url)
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
