package node

import (
	"errors"
	"fmt"
	"sync"
)

var OutOfResourcesErr = errors.New("not enough resources for function execution")

var NodeIdentifier string

type NodeResources struct {
	sync.RWMutex
	AvailableMemMB    int64
	AvailableCPUs     float64
	MaxMemMB          int64
	MaxCPUs           float64
	RequestsCount     int64   // number of requests arrived at the node
	DropRequestsCount int64   // number of requests arrived at the node but dropped in the end
	NodeExpenses      float64 // Cumulative expenses of the node in terms of $
	ContainerPools    map[string]*ContainerPool
}

func (n NodeResources) String() string {
	return fmt.Sprintf("[CPUs: %f - Mem: %d]", n.AvailableCPUs, n.AvailableMemMB)
}

var Resources NodeResources
