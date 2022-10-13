package scheduling

import "github.com/grussorusso/serverledge/internal/node"

type CustomFerraPolicy struct {
}

func (p *CustomFerraPolicy) Init() {
}

func (p *CustomFerraPolicy) OnCompletion(r *scheduledRequest) {
	//log.Printf("Completed execution of %s in %f\n", r.Fun.Name, r.ExecReport.ResponseTime)
	//Completed(r.Request, false)
}

func (p *CustomFerraPolicy) OnArrival(r *scheduledRequest) {
	dec := Decide(r)

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
	} else if dec == DROP_REQUEST {
		dropRequest(r)
	}

	/*
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
	*/

	/*
		else if r.CanDoOffloading {
			log.Println("Choosing Edge")
			url := pickEdgeNodeForOffloading(r)
			log.Printf("Choosed %s\n", url)
			if url != "" {
				handleOffload(r, url)
			} else {
				dropRequest(r)
			}
	*/
}
