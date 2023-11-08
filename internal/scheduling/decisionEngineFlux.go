package scheduling

import (
	"context"
	"log"
	"math/rand"
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

	var cFInfo *ClassFunctionInfo

	arrivalChannel <- arrivalRequest{r, class.Name}

	fInfo, prs := d.g.FunctionMap[name]
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

	/* FIXME AUDIT nContainers, _ := node.WarmStatus()[name]
	log.Printf("Function name: %s - class: %s - local node available mem: %d - func mem: %d - node containers: %d - can execute :%t - Probabilities are "+
		"\t pL: %f "+
		"\t pC: %f "+
		"\t pE: %f "+
		"\t pD: %f ", name, class.Name, node.Resources.AvailableMemMB, r.Fun.MemoryMB, nContainers, canExecute(r.Fun), pL, pC, pE, pD) */

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
				pC = 0.1
				pE = 0.9
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

	// FIXME AUDIT log.Printf("Probabilities after evaluation for %s-%s are pL:%f pE:%f pC:%f pD:%f", name, class.Name, pL, pE, pC, pD)

	// FIXME AUDIT log.Printf("prob: %f", prob)
	if prob <= pL {
		// FIXME AUDIT log.Println("Execute LOCAL")
		return LOCAL_EXEC_REQUEST
	} else if prob <= pL+pE {
		// FIXME AUDIT log.Println("Execute EDGE OFFLOAD")
		return EDGE_OFFLOAD_REQUEST
	} else if prob <= pL+pE+pC {
		// FIXME AUDIT log.Println("Execute CLOUD OFFLOAD")
		return CLOUD_OFFLOAD_REQUEST
	} else {
		// FIXME AUDIT log.Println("Execute DROP")
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
		startingLocalProb = 0.334
		startingEdgeOffloadProb = 0.333
		startingCloudOffloadProb = 0.333
	} else {
		startingLocalProb = 0.5
		startingEdgeOffloadProb = 0
		startingCloudOffloadProb = 0.5
	}

	d.g.InitMetricGrabber()

	go d.handler()
}

func (d *decisionEngineFlux) deleteOldData(period time.Duration) {
	err := deleteAPI.Delete(context.Background(), &orgServerledge, bucketServerledge, time.Now().Add(-2*period), time.Now().Add(-period), "")
	if err != nil {
		log.Println(err)
	}
}

/*
Function that:
- Handles the evaluation and calculation of the local, edge and cloud probabilities.
*/
func (d *decisionEngineFlux) handler() {
	evaluationTicker :=
		time.NewTicker(evaluationInterval)

	for {
		select {
		case _ = <-evaluationTicker.C: // Evaluation handler
			s := rand.NewSource(time.Now().UnixNano())
			rGen = rand.New(s)
			log.Println("Evaluating")

			//Check if there are some instances with 0 arrivals
			for fName, fInfo := range d.g.FunctionMap {
				for cName, cFInfo := range fInfo.invokingClasses {
					//Cleanup
					if cFInfo.arrivalCount == 0 {
						cFInfo.timeSlotsWithoutArrivals++
						if cFInfo.timeSlotsWithoutArrivals >= maxTimeSlots {
							log.Println("DELETING", fName, cName)
							d.g.Delete(fName, cName)
						}
					}
				}
			}

			//d.deleteOldData(24 * time.Hour)
			d.g.GrabMetrics()
			d.updateProbabilities()
		}
	}
}

func (d *decisionEngineFlux) updateProbabilities() {
	solve(d.g.FunctionMap)
}

func (d *decisionEngineFlux) Completed(r *scheduledRequest, offloaded int) {
	// FIXME AUDIT log.Println("COMPLETED: in decisionEngineFlux")
	d.g.Completed(r, offloaded)
}
