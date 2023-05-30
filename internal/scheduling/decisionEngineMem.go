package scheduling

import (
	"github.com/grussorusso/serverledge/internal/config"
	"log"
	"math/rand"
	"time"
)

type decisionEngineMem struct {
	m map[string]*functionInfo
}

func (d *decisionEngineMem) Decide(r *scheduledRequest) int {
	name := r.Fun.Name
	class := r.ClassService

	prob := rGen.Float64()

	var pe float64
	var po float64
	var pd float64

	var cFInfo *classFunctionInfo

	arrivalChannel <- arrivalRequest{r, class.Name}

	fInfo, prs := d.m[name]
	if !prs {
		pe = startingExecuteProb
		po = startingOffloadProb
		pd = 1 - (pe + po)
	} else {
		cFInfo, prs = fInfo.invokingClasses[class.Name]
		if !prs {
			pe = startingExecuteProb
			po = startingOffloadProb
			pd = 1 - (pe + po)
		} else {
			pe = cFInfo.probExecute
			po = cFInfo.probOffload
			pd = cFInfo.probDrop
		}
	}

	log.Println("Probabilities are", pe, po, pd)

	if !r.CanDoOffloading {
		pd = pd / (pd + pe)
		pe = pe / (pd + pe)
		po = 0
	} else if !canExecute(r.Fun) {
		if pd == 0 && po == 0 {
			pd = 0
			po = 1
			pe = 0
		} else {
			pd = pd / (pd + po)
			po = po / (pd + po)
			pe = 0
		}
	}

	if prob <= pe {
		log.Println("Execute LOCAL")
		return EXECUTE_REQUEST
	} else if prob <= pe+po {
		log.Println("Execute OFFLOAD")
		return OFFLOAD_REQUEST
	} else {
		log.Println("Execute DROP")
		return DROP_REQUEST
	}
}

func (d *decisionEngineMem) InitDecisionEngine() {
	s := rand.NewSource(time.Now().UnixNano())
	rGen = rand.New(s)

	evaluationInterval = time.Duration(config.GetInt(config.SOLVER_EVALUATION_INTERVAL, 10)) * time.Second
	log.Println("Evaluation interval:", evaluationInterval)

	d.m = make(map[string]*functionInfo)

	go d.ShowData()
	go d.handler()
}

func (d *decisionEngineMem) handler() {
	evaluationTicker :=
		time.NewTicker(evaluationInterval)
	pcoldTicker :=
		time.NewTicker(time.Duration(config.GetInt(config.CONTAINER_EXPIRATION_TIME, 600)) * time.Second)

	for {
		select {
		case _ = <-evaluationTicker.C:
			s := rand.NewSource(time.Now().UnixNano())
			rGen = rand.New(s)
			log.Println("Evaluating")

			//Check if there are some instances with 0 arrivals
			for fName, fInfo := range d.m {
				for cName, cFInfo := range fInfo.invokingClasses {
					//Cleanup
					if cFInfo.arrivalCount == 0 {
						cFInfo.timeSlotsWithoutArrivals++
						if cFInfo.timeSlotsWithoutArrivals >= maxTimeSlots {
							d.Delete(fName, cName)
						}
					}
				}
			}

			d.updateProbabilities()

			//Reset arrivals for the time slot
			for _, fInfo := range d.m {
				for _, cFInfo := range fInfo.invokingClasses {
					//Cleanup
					cFInfo.arrivalCount = 0
					cFInfo.arrivals = 0
				}
			}

		case r := <-requestChannel:
			d.updateData(r)
		case arr := <-arrivalChannel:
			name := arr.Fun.Name

			fInfo, prs := d.m[name]
			if !prs {
				fInfo = &functionInfo{
					name:            name,
					memory:          arr.Fun.MemoryMB,
					cpu:             arr.Fun.CPUDemand,
					probCold:        [2]float64{1, 1},
					invokingClasses: make(map[string]*classFunctionInfo)}

				d.m[name] = fInfo
			}

			cFInfo, prs := fInfo.invokingClasses[arr.class]
			if !prs {
				cFInfo = &classFunctionInfo{functionInfo: fInfo,
					probExecute:              startingExecuteProb,
					probOffload:              startingOffloadProb,
					probDrop:                 1 - (startingExecuteProb + startingOffloadProb),
					arrivals:                 0,
					arrivalCount:             0,
					timeSlotsWithoutArrivals: 0,
					className:                arr.class}

				fInfo.invokingClasses[arr.class] = cFInfo
			}

			cFInfo.arrivalCount++
			cFInfo.arrivals = cFInfo.arrivalCount / float64(evaluationInterval)
			cFInfo.timeSlotsWithoutArrivals = 0

		case _ = <-pcoldTicker.C:
			//Reset arrivals for the time slot
			for _, fInfo := range d.m {
				fInfo.coldStartCount = [2]int64{0, 0}
				fInfo.timeSlotCount = [2]int64{0, 0}
			}
		}
	}
}

func (d *decisionEngineMem) updateProbabilities() {
	solve(d.m)
}

func (d *decisionEngineMem) ShowData() {
	//log.Println("ERLANG: ", ErlangB(57, 45))
	//for {
	//	time.Sleep(5 * time.Second)
	//	log.Println("map", m)
	//}
	/*
		for {
			time.Sleep(5 * time.Second)
			for _, functionMap := range m {
				for _, finfo := range functionMap {
					log.Println(finfo)
				}
			}
		}
	*/
}

func (d *decisionEngineMem) Completed(r *scheduledRequest, offloaded int) {
	requestChannel <- completedRequest{
		scheduledRequest: r,
		location:         offloaded,
	}
}

func (d *decisionEngineMem) Delete(function string, class string) {
	fInfo, prs := d.m[function]
	if !prs {
		return
	}

	delete(fInfo.invokingClasses, class)

	//If there aren't any more classes calls the function can be deleted
	if len(fInfo.invokingClasses) == 0 {
		delete(d.m, function)
	}
}

/*
// TODO maybe remove
func UpdateDataAsync(r function.Response) {
	name := r.Name
	class := r.Class

	var location int

	if r.OffloadLatency != 0 {
		location = LOCAL
	} else {
		location = OFFLOADED
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

	if r.OffloadLatency != 0 {
		diff := r.OffloadLatency - OffloadLatency
		OffloadLatency = OffloadLatency +
			(1/float64(fInfo.count[location]))*(diff)
	}

	//TODO maybe remove
	if r.ResponseTime > Classes[class].MaximumResponseTime {
		fInfo.missed++
	}
}
*/

func (d *decisionEngineMem) updateData(r completedRequest) {
	name := r.Fun.Name

	location := r.location

	fInfo, prs := d.m[name]
	//TODO create here the entry in the function? Is it necessary?
	if !prs {
		return
	}

	fInfo.count[location] = fInfo.count[location] + 1
	fInfo.timeSlotCount[location] = fInfo.timeSlotCount[location] + 1

	//Welford mean and variance
	diff := r.ExecReport.Duration - fInfo.meanDuration[location]
	fInfo.meanDuration[location] = fInfo.meanDuration[location] +
		(1/float64(fInfo.count[location]))*(diff)
	diff2 := r.ExecReport.Duration - fInfo.meanDuration[location]

	fInfo.varianceDuration[location] = (diff * diff2) / float64(fInfo.count[location])

	if !r.ExecReport.IsWarmStart {
		diff := r.ExecReport.InitTime - fInfo.initTime[location]
		fInfo.initTime[location] = fInfo.initTime[location] +
			(1/float64(fInfo.count[location]))*(diff)

		fInfo.coldStartCount[location]++
		fInfo.probCold[location] = float64(fInfo.coldStartCount[location]) / float64(fInfo.timeSlotCount[location])
	}

	if r.ExecReport.OffloadLatency != 0 {
		diff := r.ExecReport.OffloadLatency - OffloadLatency
		OffloadLatency = OffloadLatency +
			(1/float64(fInfo.count[location]))*(diff)
	}
}
