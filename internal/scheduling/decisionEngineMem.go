package scheduling

import (
	"log"
)

type decisionEngineMem struct {
	g *metricGrabberMem
}

func (d *decisionEngineMem) Completed(r *scheduledRequest, offloaded int) {
	d.g.Completed(r, offloaded)
}

func (d *decisionEngineMem) GetGrabber() metricGrabber {
	return d.g
}

func (d *decisionEngineMem) Decide(r *scheduledRequest) int {
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

	log.Println("Probabilities are", pL, pC, pE, pD)

	if policyFlag == "edgeCloud" {
		if !r.CanDoOffloading {
			// Can be executed only locally or dropped
			if pL == 0 && pD == 0 && canExecute(r.Fun) {
				pL = 1
				pD = 0
				pC = 0
				pE = 0
			} else if pL == 0 && pD == 0 && !canExecute(r.Fun) {
				pL = 0
				pD = 1
				pC = 0
				pE = 0
			} else {
				pD = pD / (pD + pL)
				pL = pL / (pD + pL)
				pC = 0
				pE = 0
			}
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
		if !r.CanDoOffloading {
			// Can be executed only locally or dropped
			if pL == 0 && pD == 0 && canExecute(r.Fun) {
				pL = 1
				pD = 0
				pC = 0
				pE = 0
			} else if pL == 0 && pD == 0 && !canExecute(r.Fun) {
				pL = 0
				pD = 1
				pC = 0
				pE = 0
			} else {
				pD = pD / (pD + pL)
				pL = pL / (pD + pL)
				pC = 0
				pE = 0
			}
		} else if !canExecute(r.Fun) {
			// Node can't execute function locally
			if pD == 0 && pC == 0 && pE == 0 {
				pD = 0
				pC = 1
				pE = 0
				pL = 0
			} else {
				pD = pD / (pD + pC)
				pC = pC / (pD + pC)
				pE = 0
				pL = 0
			}
		}
	}

	if prob <= pL {
		log.Println("Execute LOCAL")
		return LOCAL_EXEC_REQUEST
	} else if prob <= pL+pC {
		log.Println("Execute CLOUD OFFLOAD")
		return CLOUD_OFFLOAD_REQUEST
	} else if prob <= pL+pC+pE && policyFlag == "edgeCloud" {
		log.Println("Execute EDGE OFFLOAD")
		return EDGE_OFFLOAD_REQUEST
	} else {
		log.Println("Execute DROP")
		requestChannel <- completedRequest{
			scheduledRequest: r,
			dropped:          true,
		}

		return DROP_REQUEST
	}
}

func (d *decisionEngineMem) InitDecisionEngine() {
	d.g.InitMetricGrabber()
}

/*
// TODO maybe remove
func UpdateDataAsync(r function.Response) {
	name := r.Name
	class := r.Class

	var location int

	if r.CloudOffloadLatency != 0 {
		location = LOCAL
	} else {
		location = OFFLOADED_CLOUD
	}

	//TODO edit this
	fInfo, prs := de.m[name]
	if !prs {
		// If it is missing from the map then enough time has passed to cause expiring on the function entry,
		// or the invocation came from somewhere else.
		// This means that maybe is not necessary to maintain information about this function
		return
	}

	fInfo.count[location] = fInfo.count[location] + 1
	fInfo.timeSlotCount[location] = fInfo.timeSlotCount[location] + 1

	//Welford mean and variance
	diff := r.Duration - fInfo.meanDuration[location]
	fInfo.meanDuration[location] = fInfo.meanDuration[location] +
		(1/float64(fInfo.count[location]))*(diff)
	diff2 := r.Duration - fInfo.meanDuration[location]

	fInfo.varianceDuration[location] = (diff * diff2) / float64(fInfo.count[location])

	if !r.IsWarmStart {
		diff := r.InitTime - fInfo.initTime[location]
		fInfo.initTime[location] = fInfo.initTime[location] +
			(1/float64(fInfo.count[location]))*(diff)

		fInfo.coldStartCount[location]++
	}

	if r.CloudOffloadLatency != 0 {
		diff := r.CloudOffloadLatency - CloudOffloadLatency
		CloudOffloadLatency = CloudOffloadLatency +
			(1/float64(fInfo.count[location]))*(diff)
	}

	//TODO maybe remove
	if r.ResponseTime > Classes[class].MaximumResponseTime {
		fInfo.missed++
	}
}
*/
