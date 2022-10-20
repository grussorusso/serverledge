package scheduling

import "github.com/grussorusso/serverledge/internal/node"

type CustomCloudOffloadPolicy struct {
}

func (p *CustomCloudOffloadPolicy) Init() {
	go InitDecisionEngine()
}

func (p *CustomCloudOffloadPolicy) OnCompletion(r *scheduledRequest) {
	//log.Printf("Completed execution of %s in %f\n", r.Fun.Name, r.ExecReport.ResponseTime)
	//Completed(r.Request, false)
}

func (p *CustomCloudOffloadPolicy) OnArrival(r *scheduledRequest) {
	dec, url := Decide(r)

	if dec == EXEC_LOCAL_REQUEST {
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
	} else if dec == EXEC_CLOUD_REQUEST {
		handleCloudOffload(r)
	} else if dec == EXEC_NEIGHBOUR_REQUEST {
		if url != "" {
			handleEdgeOffload(r, url)
			return
		}

		dropRequest(r)
	} else if dec == DROP_REQUEST {
		dropRequest(r)
	}
}
