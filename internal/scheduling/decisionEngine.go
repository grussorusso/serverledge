package scheduling

import (
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

var evaluationInterval time.Duration

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

type decisionEngine interface {
	InitDecisionEngine()
	updateProbabilities()
	Completed(r *scheduledRequest, offloaded int)
	Decide(r *scheduledRequest) int
}
