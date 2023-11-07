package scheduling

import (
	"encoding/json"
	"github.com/grussorusso/serverledge/internal/client"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/function"
	"log"
	"time"
)

/*
*

	 	Stores the function information and metrics.
		Metrics are stored as an array with size 3, to maintain also horizontal offloading data
*/
type FunctionInfo struct {
	name             string
	count            [3]int64   //Number of function requests
	meanDuration     [3]float64 //Mean duration time
	varianceDuration [3]float64 //Variance of the duration time
	coldStartCount   [3]int64   //Count the number of cold starts to estimate probCold
	timeSlotCount    [3]int64   //Count the number of calls in the time slot
	missed           int        //Number of requests that missed the deadline TODO maybe remove
	initTime         [3]float64 //Average of init times when cold start
	memory           int64      //Memory requested by the function
	cpu              float64    //CPU requested by the function
	probCold         [3]float64 //Probability of a cold start when requesting the function
	bandwidthCloud   int        //Bandwidth on cloud links
	bandwidthEdge    int        //Bandwidth on edge links
	meanInputSize    float64    //Mean size of function input

	//Map containing information about function requests for every class
	invokingClasses map[string]*ClassFunctionInfo
}

type ClassFunctionInfo struct {
	//Pointer used for accessing information about the function
	*FunctionInfo
	//
	probExecuteLocal float64
	probOffloadCloud float64
	probOffloadEdge  float64
	probDrop         float64
	//
	arrivals     float64
	arrivalCount float64
	//
	share float64
	//
	timeSlotsWithoutArrivals int
	//
	className string
}

var arrivalChannel = make(chan arrivalRequest, 1000)
var requestChannel = make(chan completedRequest, 1000)

type completedRequest struct {
	*scheduledRequest
	location int
	dropped  bool
}

type arrivalRequest struct {
	*scheduledRequest
	class string
}

var CloudOffloadLatency = 0.0 //RTT cloud
var EdgeOffloadLatency = 0.0  // RTT edge

var evaluationInterval time.Duration

func (fInfo *FunctionInfo) GetProbCold(location int) float64 {
	if fInfo.timeSlotCount[location] == 0 {
		//If there are no arrivals there's a high probability that the function execution requires a cold start
		return 1
	} else {
		return float64(fInfo.coldStartCount[location]) / float64(fInfo.timeSlotCount[location])
	}
}

func estimateLatency(r *scheduledRequest) (float64, float64) {
	// Execute a type assertion to access FunctionMap
	var fun *FunctionInfo
	if flux, ok := grabber.(*metricGrabberFlux); ok {
		// Access function map
		fun = flux.FunctionMap[r.Fun.Name]
		if fun == nil {
			return latencyLocal, latencyCloud
		}
	}
	latencyLocal = fun.meanDuration[0] + fun.probCold[0]*fun.initTime[0]
	latencyCloud = fun.meanDuration[1] + fun.probCold[1]*fun.initTime[1] +
		2*CloudOffloadLatency +
		fun.meanInputSize*8/1000/1000/config.GetFloat(config.BANDWIDTH_CLOUD, 1.0)
	log.Println("Latency local: ", latencyLocal)
	log.Println("Latency cloud: ", latencyCloud)
	return latencyLocal, latencyCloud
}

func CalculatePacketSize(r *function.Request) int {
	request := client.InvocationRequest{Params: r.Params,
		QoSMaxRespT: r.MaxRespT,
		Async:       r.Async}
	invocationBody, err := json.Marshal(request)
	if err != nil {
		log.Print(err)
	}
	// Calculate approximate packet size
	sizePacket := len(invocationBody)
	log.Println("size packet calculated")
	return sizePacket
}

type metricGrabber interface {
	InitMetricGrabber()
	GrabMetrics()
	Completed(r *scheduledRequest, offloaded int)
	Delete(function string, class string)
}
