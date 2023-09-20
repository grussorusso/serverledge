package scheduling

import (
	"encoding/json"
	"fmt"
	"github.com/grussorusso/serverledge/internal/client"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
	"log"
	"math/rand"
	"time"
)

const (
	LOCAL           = 0
	OFFLOADED_CLOUD = 1
	OFFLOADED_EDGE  = 2
)

const (
	DROP_REQUEST          = 0
	LOCAL_EXEC_REQUEST    = 1
	CLOUD_OFFLOAD_REQUEST = 2
	EDGE_OFFLOAD_REQUEST  = 3
)

var startingLocalProb = 0.3        //0.0
var startingCloudOffloadProb = 0.3 //1.0
var startingEdgeOffloadProb = 0.3

var evaluationInterval time.Duration

var CloudOffloadLatency = 0.0 //RTT cloud
var EdgeOffloadLatency = 0.0  // RTT edge

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
	packetSize       int        //Size of the function packet to send to possible host

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

var rGen *rand.Rand

var arrivalChannel = make(chan arrivalRequest, 1000)
var requestChannel = make(chan completedRequest, 1000)

// TODO add to config
var maxTimeSlots = 20

type completedRequest struct {
	*scheduledRequest
	location int
	dropped  bool
}

type arrivalRequest struct {
	*scheduledRequest
	class string
}

// TODO remove
func (fInfo *functionInfo) getProbCold(location int) float64 {
	if fInfo.timeSlotCount[location] == 0 {
		//If there are no arrivals there's a high probability that the function execution requires a cold start
		return 1
	} else {
		return float64(fInfo.coldStartCount[location]) / float64(fInfo.timeSlotCount[location])
	}
}

func canExecute(function *function.Function) bool {
	nContainers, _ := node.WarmStatus()[function.Name]
	if nContainers >= 1 {
		return true
	}

	if node.Resources.AvailableCPUs >= function.CPUDemand &&
		node.Resources.AvailableMemMB >= function.MemoryMB {
		return true
	}

	return false
}

func recoverRemoteUrl(r *scheduledRequest, isCloudUrl bool) string {
	if isCloudUrl {
		return config.GetString(config.CLOUD_URL, "")
	} else {
		// pick the edge node used for offloading and recover its url
		return pickEdgeNodeForOffloading(r)
	}
}

func calculatePacketSize(r *scheduledRequest, isCloudCalc bool) int {
	serverUrl := recoverRemoteUrl(r, isCloudCalc)
	request := client.InvocationRequest{Params: r.Params,
		QoSMaxRespT: r.MaxRespT,
		Async:       r.Async}
	invocationBody, err := json.Marshal(request)
	if err != nil {
		log.Print(err)
	}

	// Calculate approximate packet size
	sizePacket := len("POST /invoke/"+r.Fun.Name+" HTTP/1.1") +
		len("Host: "+serverUrl) +
		len("User-Agent: Go-http-client/1.1") +
		len("Content-Length: "+fmt.Sprintf("%d", len(invocationBody))) +
		len("Content-Type: application/json") +
		len("\r\n\r\n") +
		len(invocationBody)
	log.Println("size packet calculated")
	return sizePacket
}

type decisionEngine interface {
	InitDecisionEngine()
	updateProbabilities()
	Completed(r *scheduledRequest, offloaded int)
	Decide(r *scheduledRequest) int
	Delete(function string, class string)
}
