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
	AvailableMemMB int64
	AvailableCPUs  float64
	DropCount      int64
	ContainerPools map[string]*ContainerPool
}

func (n *NodeResources) String() string {
	return fmt.Sprintf("[CPUs: %f - Mem: %d]", n.AvailableCPUs, n.AvailableMemMB)
}

var Resources NodeResources
