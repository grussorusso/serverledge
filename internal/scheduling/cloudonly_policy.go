package scheduling

type CloudOnlyPolicy struct{}

func (p *CloudOnlyPolicy) Init() {
}

func (p *CloudOnlyPolicy) OnCompletion(r *scheduledRequest) {

}

func (p *CloudOnlyPolicy) OnArrival(r *scheduledRequest) {
	if r.CanDoOffloading {
		handleCloudOffload(r)
	} else {
		dropRequest(r)
	}
}

func (p *CloudOnlyPolicy) OnRestore(r *scheduledRestore) {
	containerID, error := Restore(r.contID, r.archiveName)
	restoreResponse := &restoreResult{
		contID: containerID,
		err:    error,
	}
	r.restoreChannel <- *restoreResponse
}
