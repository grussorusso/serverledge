package containers

import (
	"container/list"
	"errors"
	"sync"
)

type functionPool struct {
	sync.Mutex
	busy  *list.List // list of ContainerID
	ready *list.List // list of warmContainer
}

type warmContainer struct {
	Expiration int64
	contID     ContainerID
}

type NodeResources struct {
	sync.Mutex
	AvailableMemMB int64
	AvailableCPUs  float64
}

var OutOfResourcesErr = errors.New("Not enough resources for function execution")
var NoWarmFoundErr = errors.New("No warm container is available.")
