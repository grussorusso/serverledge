package node

import (
	"errors"
	"sync"
)

var OutOfResourcesErr = errors.New("not enough resources for function execution")

type NodeResources struct {
	sync.RWMutex
	AvailableMemMB int64
	AvailableCPUs  float64
	DropCount      int64
	ContainerPools map[string]*ContainerPool
}

var Resources NodeResources
