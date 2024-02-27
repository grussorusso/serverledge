package scheduling

import (
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/node"
	"log"
)

var de decisionEngine

type CustomCloudOffloadPolicy struct {
}

// TODO add configuration for different types of decision engines
func (p *CustomCloudOffloadPolicy) Init() {
	version := config.GetString(config.STORAGE_VERSION, "flux")
	if version == "mem" {
		// fixme ADD METRIC GRABBER MEM NON WORKING NOW
		de = &decisionEngineMem{}
	} else {
		de = &decisionEngineFlux{
			&metricGrabberFlux{},
		}
	}

	log.Println("Policy version:", version)

	de.InitDecisionEngine()
}

// TODO move completed jobs here
func (p *CustomCloudOffloadPolicy) OnCompletion(r *scheduledRequest) {
	//log.Printf("Completed execution of %s in %f\n", r.Fun.Name, r.ExecReport.ResponseTime)
	//Completed(r.Request, false)
	if r.ExecReport.SchedAction == SCHED_ACTION_OFFLOAD_CLOUD {
		de.Completed(r, OFFLOADED_CLOUD)
	} else {
		de.Completed(r, LOCAL)
	}
}

func (p *CustomCloudOffloadPolicy) OnArrival(r *scheduledRequest) {
	dec := de.Decide(r)

	if dec == LOCAL_EXEC_REQUEST {
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
	} else if dec == CLOUD_OFFLOAD_REQUEST {
		handleCloudOffload(r)
	} else if dec == DROP_REQUEST {
		dropRequest(r)
	}
}
