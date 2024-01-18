package scheduling

import (
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
)

// Custom1Policy executes locally if possible, otherwise if the local resource are not enough:
//
// - chooses an edge node when a function has a HIGH_PERFORMANCE class
//
// - chooses a cloud node if offloading is enabled
//
// - drops the request if offloading is disabled and previous conditions are not met
type Custom1Policy struct {
}

func (p *Custom1Policy) Init() {
}

func (p *Custom1Policy) OnCompletion(r *scheduledRequest) {

}

func (p *Custom1Policy) OnArrival(r *scheduledRequest) {

	containerID, err := node.AcquireWarmContainer(r.Fun)
	if err == nil {
		execLocally(r, containerID, true)
	} else if handleColdStart(r) {
		return
	} else if r.CanDoOffloading && r.RequestQoS.Class == function.HIGH_PERFORMANCE {
		url := pickEdgeNodeForOffloading(r)
		if url != "" {
			handleOffload(r, url)
		} else {
			dropRequest(r)
		}
	} else if r.CanDoOffloading {
		handleCloudOffload(r)
	} else {
		dropRequest(r)
	}
}
