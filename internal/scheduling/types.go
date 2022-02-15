package scheduling

import (
	"container/list"
	"errors"
	"sync"

	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/function"
)

type containerPool struct {
	sync.Mutex
	busy  *list.List // list of ContainerID
	ready *list.List // list of warmContainer
}

type warmContainer struct {
	Expiration int64
	contID     container.ContainerID
}

type NodeResources struct {
	sync.RWMutex
	AvailableMemMB int64
	AvailableCPUs  float64
	containerPools map[string]*containerPool
}

var OutOfResourcesErr = errors.New("not enough resources for function execution")
var NoWarmFoundErr = errors.New("no warm container is available")

// scheduledRequest represents a Request within the scheduling subsystem
type scheduledRequest struct {
	*function.Request
	decisionChannel chan schedDecision
}

// schedDecision wraps a action made by the scheduler.
// Possible decisions are 1) drop, 2) execute locally or 3) execute on a remote
// node (offloading).
type schedDecision struct {
	action     action
	contID     container.ContainerID
	remoteHost string
}

type action int64

const (
	DROP                  action = 0
	EXEC_LOCAL                   = 1
	EXEC_REMOTE                  = 2
	BEST_EFFORT_EXECUTION        = 3
)

type schedulingDecision int64

const (
	SCHED_DROP   schedulingDecision = 0
	SCHED_REMOTE                    = 1
	SCHED_LOCAL                     = 2
	SCHED_BASIC                     = 3
)
