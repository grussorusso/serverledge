package scheduling

import (
	"encoding/json"
	"github.com/grussorusso/serverledge/internal/client"
	"github.com/grussorusso/serverledge/internal/function"
	"log"
	"time"
)

/*
*

	 	Stores the function information and metrics.
		Metrics are stored as an array with size 3, to maintain also horizontal offloading data
*/
type functionInfo struct {
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
	invokingClasses map[string]*classFunctionInfo
}

type classFunctionInfo struct {
	//Pointer used for accessing information about the function
	*functionInfo
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

func (fInfo *functionInfo) GetProbCold(location int) float64 {
	if fInfo.timeSlotCount[location] == 0 {
		//If there are no arrivals there's a high probability that the function execution requires a cold start
		return 1
	} else {
		return float64(fInfo.coldStartCount[location]) / float64(fInfo.timeSlotCount[location])
	}
}

// CalculatePacketSize Calculates the packet size of the request (used to estimate latency between nodes).
func CalculatePacketSize(r *function.Request) int {
	invocationBody, err := json.Marshal(client.InvocationRequest{
		Params:      r.Params,
		QoSMaxRespT: r.MaxRespT,
		Async:       r.Async,
	})
	if err != nil {
		log.Println(err)
		return 0
	}

	return len(invocationBody)
}

type metricGrabber interface {
	InitMetricGrabber()
	GrabFunctionInfo(functionName string) (*functionInfo, bool)
	Completed(r *scheduledRequest, offloaded int)
	Delete(function string, class string)
	updateProbabilities()
	queryMetrics()
}
