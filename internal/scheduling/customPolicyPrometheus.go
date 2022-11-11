package scheduling

import "github.com/grussorusso/serverledge/internal/node"

type CustomPolicyPrometheus struct {
}

var dP *decisionEnginePrometheus

func (p *CustomPolicyPrometheus) Init() {
	dP = &decisionEnginePrometheus{}
	go d.InitDecisionEngine()
}

func (p *CustomPolicyPrometheus) OnCompletion(r *scheduledRequest) {

}

func (p *CustomPolicyPrometheus) OnArrival(r *scheduledRequest) {
	dec := dP.Decide(r)

	if dec == EXECUTE_REQUEST {
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
	} else if dec == OFFLOAD_REQUEST {
		handleCloudOffload(r)
	} else if dec == DROP_REQUEST {
		dropRequest(r)
	}
}
