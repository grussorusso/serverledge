package scheduling

import (
	"context"
	"log"
	"time"
)

type decisionEngineFlux struct {
	g *metricGrabberFlux
}

func (d *decisionEngineFlux) Decide(r *scheduledRequest) int {
	name := r.Fun.Name
	class := r.ClassService

	prob := rGen.Float64()

	var pL float64
	var pC float64
	var pE float64
	var pD float64

	var cFInfo *classFunctionInfo

	arrivalChannel <- arrivalRequest{r, class.Name}

	fInfo, prs := d.g.GrabFunctionInfo(name)
	if !prs {
		pL = startingLocalProb
		pC = startingCloudOffloadProb
		pE = startingEdgeOffloadProb
		pD = 1 - (pL + pC + pE)
	} else {
		cFInfo, prs = fInfo.invokingClasses[class.Name]
		if !prs {
			pL = startingLocalProb
			pC = startingCloudOffloadProb
			pE = startingEdgeOffloadProb
			pD = 1 - (pL + pC + pE)
		} else {
			pL = cFInfo.probExecuteLocal
			pC = cFInfo.probOffloadCloud
			pE = cFInfo.probOffloadEdge
			pD = cFInfo.probDrop
		}
	}

	/*nContainers, _ := node.WarmStatus()[name]
	log.Printf("Function name: %s - class: %s - local node available mem: %d - func mem: %d - node containers: %d - can execute :%t - Probabilities are: \n"+
		"\t pL: %f \n"+
		"\t pC: %f \n"+
		"\t pE: %f \n"+
		"\t pD: %f \n", name, class.Name, node.Resources.AvailableMemMB, r.Fun.MemoryMB, nContainers, canExecute(r.Fun), pL, pC, pE, pD)*/

	if policyFlag == "edgeCloud" {
		// Cloud and Edge offloading allowed
		if !r.CanDoOffloading {
			// Can be executed only locally or dropped
			pD = pD / (pD + pL)
			pL = pL / (pD + pL)
			pC = 0
			pE = 0
		} else if !canExecute(r.Fun) {
			// Node can't execute function locally
			if pD == 0 && pC == 0 && pE == 0 {
				pD = 0
				pC = 0.5
				pE = 0.5
				pL = 0
			} else {
				pD = pD / (pD + pC + pE)
				pC = pC / (pD + pC + pE)
				pE = pE / (pD + pC + pE)
				pL = 0
			}
		}
	} else {
		// Cloud only
		if !r.CanDoOffloading {
			pD = pD / (pD + pL)
			pL = pL / (pD + pL)
			pC = 0
			pE = 0
		} else if !canExecute(r.Fun) {
			if pD == 0 && pC == 0 {
				// Node can't execute function locally
				pD = 0
				pE = 0
				pC = 1
				pL = 0
			} else {
				pD = pD / (pD + pC)
				pC = pC / (pD + pC)
				pE = 0
				pL = 0
			}
		}
	}

	if !canAffordCloudOffloading(r) {
		pC = 0
		if pL == 0 && pE == 0 && pD == 0 {
			pL = 0
			pE = 0
			pD = 1
		} else {
			pL = pL / (pL + pE + pD)
			pE = pE / (pL + pE + pD)
			pD = pD / (pL + pE + pD)
		}
	}

	//log.Printf("Probabilities after evaluation for %s-%s are pL:%f pE:%f pC:%f pD:%f", name, class.Name, pL, pE, pC, pD)

	//log.Printf("prob: %f", prob)
	if prob <= pL {
		//log.Println("Execute LOCAL")
		return LOCAL_EXEC_REQUEST
	} else if prob <= pL+pE {
		//log.Println("Execute EDGE OFFLOAD")
		return EDGE_OFFLOAD_REQUEST
	} else if prob <= pL+pE+pC {
		//log.Println("Execute CLOUD OFFLOAD")
		return CLOUD_OFFLOAD_REQUEST
	} else {
		//log.Println("Execute DROP")
		requestChannel <- completedRequest{
			scheduledRequest: r,
			dropped:          true,
		}

		return DROP_REQUEST
	}
}

func (d *decisionEngineFlux) InitDecisionEngine() {
	// Initializing starting probabilities
	if policyFlag == "edgeCloud" {
		startingLocalProb = 0.5
		startingEdgeOffloadProb = 0.25
		startingCloudOffloadProb = 0.25
	} else {
		startingLocalProb = 0.5
		startingEdgeOffloadProb = 0
		startingCloudOffloadProb = 0.5
	}
	startTime = time.Now()

	d.g.InitMetricGrabber()
}

func (d *decisionEngineFlux) deleteOldData(period time.Duration) {
	err := deleteAPI.Delete(context.Background(), &orgServerledge, bucketServerledge, time.Now().Add(-2*period), time.Now().Add(-period), "")
	if err != nil {
		log.Println(err)
	}
}

func (d *decisionEngineFlux) Completed(r *scheduledRequest, offloaded int) {
	// FIXME AUDIT log.Println("COMPLETED: in decisionEngineFlux")
	d.g.Completed(r, offloaded)
}

func (d *decisionEngineFlux) GetGrabber() metricGrabber {
	return d.g
}
