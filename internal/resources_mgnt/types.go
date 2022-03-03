package resources_mgnt

import (
	"container/list"
	"errors"
	"github.com/grussorusso/serverledge/internal/container"
	"sync"
)

type ContainerPool struct {
	//	sync.Mutex
	busy  *list.List // list of ContainerID
	ready *list.List // list of warmContainer
}

type warmContainer struct {
	Expiration int64
	contID     container.ContainerID
}

var OutOfResourcesErr = errors.New("not enough resources for function execution")
var NoWarmFoundErr = errors.New("no warm container is available")

type NodeResources struct {
	sync.RWMutex
	AvailableMemMB int64
	AvailableCPUs  float64
	DropCount      int64
	ContainerPools map[string]*ContainerPool
}
