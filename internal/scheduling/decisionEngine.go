package scheduling

import (
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
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

var startingLocalProb = 0.5         //Optimistically start with a higher probability of executing function locally
var startingCloudOffloadProb = 0.25 //
var startingEdgeOffloadProb = 0.25  // It's equally probable that we have a vertical offload and a horizontal offload
var startTime time.Time             // Initial timestamp of the execution

var rGen *rand.Rand

// TODO add to config
var maxTimeSlots = 20

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

// CalculateExpectedCost Calculates the expected cost of a scheduled request. It's used to check if the node can afford Cloud offloading
func CalculateExpectedCost(r *scheduledRequest) float64 {
	fInfo, prs := engine.GetGrabber().GrabFunctionInfo(r.Fun.Name)
	if !prs {
		return 0
	}
	return config.GetFloat(config.CLOUD_COST_FACTOR, 0.01) * fInfo.meanDuration[2] * (float64(r.Fun.MemoryMB) / 1024)
}

func canAffordCloudOffloading(r *scheduledRequest) bool {
	// Need to check if I can financially afford to offload to Cloud node
	executionTime := time.Now().Sub(startTime).Seconds()
	localBudget := config.GetFloat(config.BUDGET, 0.01)
	meanExpense := (node.Resources.NodeExpenses + CalculateExpectedCost(r)) / executionTime * 3600
	//log.Println("localBudget: ", localBudget)
	//log.Println("totalExpense: ", node.Resources.NodeExpenses/executionTime)
	//log.Println("expectedExpense: ", meanExpense)
	if meanExpense > localBudget {
		//log.Printf("Cannot afford Cloud - dropping request")
		return false
	} else {
		//log.Printf("Can afford Cloud - proceeding with vertical offloading")
		return true
	}
}

type decisionEngine interface {
	InitDecisionEngine()
	Completed(r *scheduledRequest, offloaded int)
	Decide(r *scheduledRequest) int
	GetGrabber() metricGrabber
}
