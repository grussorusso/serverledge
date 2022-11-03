package scheduling

import (
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
	"log"
	"math/rand"
	"time"
)

const (
	LOCAL     = 0
	OFFLOADED = 1
)

const (
	DROP_REQUEST    = 0
	EXECUTE_REQUEST = 1
	OFFLOAD_REQUEST = 2
)

var startingExecuteProb = 0.5
var startingOffloadProb = 0.5

// TODO add to config
var evaluationInterval = 10
var maxTimeSlots = 5

var rGen *rand.Rand

type classFunctionInfo struct {
	*functionInfo
	//
	probExecute float64
	probOffload float64
	probDrop    float64
	//
	probCold float64
	//
	arrivals     float64
	arrivalCount float64
	//
	share float64
	//
	timeSlotsWithoutArrivals int
}

type functionInfo struct {
	name string
	//Number of function requests
	count [2]int
	//Mean duration time
	meanDuration [2]float64
	//Variance of the duration time
	varianceDuration [2]float64
	//Number of requests that missed the deadline
	missed int
	//Offload latency
	offloadTime float64
	//Average of init times when cold start
	initTime float64
	//
	memory int64
	cpu    float64
	//
	probCold float64
	//
	invokingClasses map[string]*classFunctionInfo
}

type completedRequest struct {
	*function.Request
	location int
}

type arrivalRequest struct {
	*scheduledRequest
	class string
}

var m = make(map[string]*functionInfo)

// TODO edit buffer?
var arrivalChannel = make(chan arrivalRequest, 1000)
var requestChannel = make(chan completedRequest, 1000)

func Decide(r *scheduledRequest) int {
	name := r.Fun.Name
	class := r.ClassService

	prob := rGen.Float64()

	var pe float64
	var po float64
	var pd float64

	var cFInfo *classFunctionInfo

	arrivalChannel <- arrivalRequest{r, class.Name}

	fInfo, prs := m[name]
	if !prs {
		pe = startingExecuteProb
		po = startingOffloadProb
		pd = 1 - (pe + po)
	} else {
		log.Println("PCold", fInfo.probCold)
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

	//warmNumber, isWarm := node.WarmStatus()[name]
	if !r.CanDoOffloading {
		pd = pd / (pd + pe)
		pe = pe / (pd + pe)
		po = 0
	} else if node.Resources.AvailableCPUs < r.Fun.CPUDemand &&
		node.Resources.AvailableMemMB < r.Fun.MemoryMB {
		pd = pd / (pd + po)
		po = po / (pd + po)
		pe = 0
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

func InitDecisionEngine() {
	s := rand.NewSource(time.Now().UnixNano())
	rGen = rand.New(s)

	go ShowData()
	go handler()
}

func handler() {
	evaluationTicker :=
		time.NewTicker(time.Duration(evaluationInterval) * time.Second)

	for {
		select {
		case _ = <-evaluationTicker.C:
			s := rand.NewSource(time.Now().UnixNano())
			rGen = rand.New(s)
			log.Println("Evaluating")

			//Check if there are some instances with 0 arrivals
			for fName, fInfo := range m {
				for cName, cFInfo := range fInfo.invokingClasses {
					//Cleanup
					if cFInfo.arrivalCount == 0 {
						cFInfo.timeSlotsWithoutArrivals++
						if cFInfo.timeSlotsWithoutArrivals >= maxTimeSlots {
							Delete(fName, cName)
						}
					}
				}
			}

			updateProbabilities()

			//Reset arrivals for the time slot
			for _, fInfo := range m {
				for _, cFInfo := range fInfo.invokingClasses {
					//Cleanup
					cFInfo.arrivalCount = 0
					cFInfo.arrivals = 0
				}
			}

		case r := <-requestChannel:
			updateData(r)
		case arr := <-arrivalChannel:
			name := arr.Fun.Name

			fInfo, prs := m[name]
			if !prs {
				fInfo = &functionInfo{
					name:            name,
					memory:          arr.Fun.MemoryMB,
					cpu:             arr.Fun.CPUDemand,
					probCold:        1,
					invokingClasses: make(map[string]*classFunctionInfo)}

				m[name] = fInfo
			}

			cFInfo, prs := fInfo.invokingClasses[arr.class]
			if !prs {
				cFInfo = &classFunctionInfo{functionInfo: fInfo,
					probExecute:              startingExecuteProb,
					probOffload:              startingOffloadProb,
					probDrop:                 1 - (startingExecuteProb + startingOffloadProb),
					arrivals:                 0,
					arrivalCount:             0,
					timeSlotsWithoutArrivals: 0}

				fInfo.invokingClasses[arr.class] = cFInfo
			}

			cFInfo.arrivalCount++
			cFInfo.arrivals = cFInfo.arrivalCount / float64(evaluationInterval)
			cFInfo.timeSlotsWithoutArrivals = 0
		}
	}
}

func updateProbabilities() {
	SolveProbabilities(m)
	log.Println(SolveColdStart(m))
}

func ShowData() {
	//log.Println("ERLANG: ", ErlangB(57, 45))
	for {
		time.Sleep(5 * time.Second)
		log.Println("map", m)
	}
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

func Completed(r *function.Request, offloaded int) {
	requestChannel <- completedRequest{
		Request:  r,
		location: offloaded,
	}
}

func Delete(function string, class string) {
	fInfo, prs := m[function]
	if !prs {
		return
	}

	delete(fInfo.invokingClasses, class)

	//If there aren't any more classes calls the function can be deleted
	if len(fInfo.invokingClasses) == 0 {
		delete(m, function)
	}
}

func UpdateDataAsync(r function.Response) {
	name := r.Name
	class := r.Class

	var location int

	if r.OffloadLatency != 0 {
		location = LOCAL
	} else {
		location = OFFLOADED
	}

	fInfo, prs := m[name]
	if !prs {
		//log.Fatal("MISSING FUNCTION INFO")
		// If it is missing from the map then enough time has passed to cause expiring on the function entry,
		// or the invocation came from somewhere else.
		// This means that maybe is not necessary to maintain information about this function
		return
	}

	fInfo.count[location] = fInfo.count[location] + 1

	//Welford mean and variance
	diff := r.Duration - fInfo.meanDuration[location]
	fInfo.meanDuration[location] = fInfo.meanDuration[location] +
		(1/float64(fInfo.count[location]))*(diff)
	diff2 := r.Duration - fInfo.meanDuration[location]

	fInfo.varianceDuration[location] = (diff * diff2) / float64(fInfo.count[location])

	if !r.IsWarmStart {
		diff := r.InitTime - fInfo.initTime
		fInfo.initTime = fInfo.initTime +
			(1/float64(fInfo.count[location]))*(diff)
	}

	if r.OffloadLatency != 0 {
		diff := r.OffloadLatency - fInfo.offloadTime
		fInfo.offloadTime = fInfo.offloadTime +
			(1/float64(fInfo.count[location]))*(diff)
	}

	//TODO maybe remove
	if r.ResponseTime > Classes[class].MaximumResponseTime {
		fInfo.missed++
	}
}

func updateData(r completedRequest) {
	name := r.Fun.Name

	location := r.location

	fInfo, prs := m[name]
	if !prs {
		log.Fatal("MISSING FUNCTION INFO")
		return
	}

	fInfo.count[location] = fInfo.count[location] + 1

	//Welford mean and variance
	diff := r.ExecReport.Duration - fInfo.meanDuration[location]
	fInfo.meanDuration[location] = fInfo.meanDuration[location] +
		(1/float64(fInfo.count[location]))*(diff)
	diff2 := r.ExecReport.Duration - fInfo.meanDuration[location]

	fInfo.varianceDuration[location] = (diff * diff2) / float64(fInfo.count[location])

	if !r.ExecReport.IsWarmStart {
		diff := r.ExecReport.InitTime - fInfo.initTime
		fInfo.initTime = fInfo.initTime +
			(1/float64(fInfo.count[location]))*(diff)
	}

	if r.ExecReport.OffloadLatency != 0 {
		diff := r.ExecReport.OffloadLatency - fInfo.offloadTime
		fInfo.offloadTime = fInfo.offloadTime +
			(1/float64(fInfo.count[location]))*(diff)
	}

	if r.ExecReport.ResponseTime > r.GetMaxRT() {
		fInfo.missed++
	}
}
