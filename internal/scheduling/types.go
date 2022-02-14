package scheduling

import (
	"container/list"
	"errors"
	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/function"
	"sync"
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
	sync.Mutex
	AvailableMemMB int64
	AvailableCPUs  float64
	containerPools map[string]*containerPool
}

var OutOfResourcesErr = errors.New("Not enough resources for function execution")
var NoWarmFoundErr = errors.New("No warm container is available.")

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
	DROP        action = 0
	EXEC_LOCAL         = 1
	EXEC_REMOTE        = 2
)
