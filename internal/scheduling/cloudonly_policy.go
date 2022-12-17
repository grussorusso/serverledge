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
	resoreResponse := &restoreResult{
		err: Restore(r.contID, r.archiveName),
	}
	r.restoreChannel <- *resoreResponse
}
