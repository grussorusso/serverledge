package scheduling

import (
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
)

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
			//handleOffload(r, url)
		} else {
			dropRequest(r)
		}
	} else if r.CanDoOffloading {
		handleCloudOffload(r)
	} else {
		dropRequest(r)
	}
}
