package scheduling

import (
	"github.com/grussorusso/serverledge/internal/config"
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
	if r.ExecReport.SchedAction == SCHED_ACTION_OFFLOAD_CLOUD {
		engine.Completed(r, OFFLOADED_CLOUD)
	} else if r.ExecReport.SchedAction == SCHED_ACTION_OFFLOAD_EDGE {
		engine.Completed(r, OFFLOADED_EDGE)
	} else {
		engine.Completed(r, LOCAL)
	}
}

func (p *OffloadToCluster) OnArrival(r *scheduledRequest) {
	dec := engine.Decide(r)

	if dec == LOCAL_EXEC_REQUEST {
		containerID, err := node.AcquireWarmContainer(r.Fun)
		if err == nil {
			log.Printf("Using a warm container for: %v", r)
			execLocally(r, containerID, true)
		} else if handleColdStart(r) {
			log.Printf("No warm containers for: %v - COLD START", r)
			return
		} else if r.CanDoOffloading {
			// horizontal offloading - search for a nearby node to offload
			log.Printf("No warm containers and node cant'handle cold start due to lack of resources: proceeding with offloading")
			url := pickEdgeNodeForOffloading(r)
			if url != "" {
				log.Printf("Found node at url: %s - proceeding with horizontal offloading", url)
				handleEdgeOffload(r, url)
			} else {
				log.Printf("Cant find nearby nodes - proceeding with vertical offloading")
				handleCloudOffload(r)
			}
		} else {
			log.Printf("Can't execute locally and can't offload - dropping incoming request")
			dropRequest(r)
		}
	} else if dec == CLOUD_OFFLOAD_REQUEST {
		handleCloudOffload(r)
		/*if r.RequestQoS.Class == function.HIGH_PERFORMANCE {
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
		}*/
	} else if dec == EDGE_OFFLOAD_REQUEST {
		log.Println("DEC IS EDGE_OFFLOAD_REQUEST")
		url := pickEdgeNodeForOffloading(r)
		if url != "" {
			handleEdgeOffload(r, url)
		} else {
			log.Println("Can't execute horizontal offloading due to lack of resources available: offloading to cloud")
			handleCloudOffload(r)
		}
	} else if dec == DROP_REQUEST {
		dropRequest(r)
	}
}
