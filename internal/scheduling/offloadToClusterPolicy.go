package scheduling

import (
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
	"log"
)

// EdgePolicy supports only Edge-Edge offloading

type OffloadToCluster struct{}

var engine decisionEngine

func (p *OffloadToCluster) Init() {
	// initialize decision engine
	version := config.GetString(config.SCHEDULING_POLICY_VERSION, "flux")
	if version == "mem" {
		engine = &decisionEngineMem{}
	} else {
		engine = &decisionEngineFlux{}
	}
	log.Println("Policy version:", version)
	engine.InitDecisionEngine()
}

func (p *OffloadToCluster) OnCompletion(r *scheduledRequest) {
	if r.ExecReport.SchedAction == SCHED_ACTION_OFFLOAD {
		engine.Completed(r, OFFLOADED)
	} else {
		engine.Completed(r, LOCAL)
	}
}

func (p *OffloadToCluster) OnArrival(r *scheduledRequest) {
	dec := engine.Decide(r)

	if dec == EXECUTE_REQUEST {
		containerID, err := node.AcquireWarmContainer(r.Fun)
		if err == nil {
			log.Printf("Using a warm container for: %v", r)
			execLocally(r, containerID, true)
			return
		} else if handleColdStart(r) {
			log.Printf("No warm containers for: %v - COLD START", r)
			return
		} else if r.CanDoOffloading {
			// horizontal offloading - search for a nearby node to offload
			log.Printf("Picking edge node for horizontal offloading")
			url := pickEdgeNodeForOffloading(r)
			if url != "" {
				log.Printf("Found url: %s", url)
				handleEdgeOffload(r, url)
				return
			} else {
				handleCloudOffload(r)
			}
		} else {
			dropRequest(r)
		}
	} else if dec == OFFLOAD_REQUEST {
		if r.RequestQoS.Class == function.HIGH_PERFORMANCE {
			// The function needs high performances, offload to cloud
			log.Printf("Offloading to cloud")
			handleCloudOffload(r)
			return
		} else {
			// The function doesn't need high performances, offload to edge node
			log.Printf("Picking edge node for horizontal offloading")
			url := pickEdgeNodeForOffloading(r)
			if url != "" {
				log.Printf("Found url: %s", url)
				handleEdgeOffload(r, url)
			} else {
				handleCloudOffload(r)
			}
			return
		}
	} else if dec == DROP_REQUEST {
		dropRequest(r)
	}
}
