package containers

import (
	"container/list"
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
