package scheduling

type CloudOnlyPolicy struct{}

func (p *CloudOnlyPolicy) Init() {
}

func (p *CloudOnlyPolicy) OnCompletion(_ *scheduledRequest) {

}

func (p *CloudOnlyPolicy) OnArrival(r *scheduledRequest) {
	if r.CanDoOffloading {
		handleCloudOffload(r)
	} else {
		dropRequest(r)
	}
}
