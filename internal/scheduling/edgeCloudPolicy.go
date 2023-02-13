package scheduling

import (
	"github.com/grussorusso/serverledge/internal/node"
)

// CloudEdgePolicy supports only Edge-Cloud Offloading
type CloudEdgePolicy struct{}

func (p *CloudEdgePolicy) Init() {
}

func (p *CloudEdgePolicy) OnCompletion(r *scheduledRequest) {

}

func (p *CloudEdgePolicy) OnArrival(r *scheduledRequest) {
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
}

func (p *CloudEdgePolicy) OnRestore(r *scheduledRestore) {
	containerID, error := Restore(r.contID, r.archiveName)
	restoreResponse := &restoreResult{
		contID: containerID,
		err:    error,
	}
	r.restoreChannel <- *restoreResponse
}
