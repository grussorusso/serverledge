package node

import (
	"container/list"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/grussorusso/serverledge/internal/container"

	"github.com/grussorusso/serverledge/internal/config"

	"github.com/grussorusso/serverledge/internal/function"
)

type ContainerPool struct {
	busy  *list.List // list of ContainerID
	ready *list.List // list of warmContainer
}

type warmContainer struct {
	Expiration int64
	contID     container.ContainerID
}

var NoWarmFoundErr = errors.New("no warm container is available")

//getFunctionPool retrieves (or creates) the container pool for a function.
func getFunctionPool(f *function.Function) *ContainerPool {
	if fp, ok := Resources.ContainerPools[f.Name]; ok {
		return fp
	}

	fp := newFunctionPool(f)
	Resources.ContainerPools[f.Name] = fp
	return fp
}

func (fp *ContainerPool) acquireReadyContainer() (container.ContainerID, bool) {
	// TODO: picking most-recent / least-recent container might be better?
	elem := fp.ready.Front()
	if elem == nil {
		return "", false
	}

	fp.ready.Remove(elem)
	contID := elem.Value.(warmContainer).contID
	fp.putBusyContainer(contID)

	return contID, true
}

func (fp *ContainerPool) putBusyContainer(contID container.ContainerID) {
	fp.busy.PushBack(contID)
}

func (fp *ContainerPool) putReadyContainer(contID container.ContainerID, expiration int64) {
	fp.ready.PushBack(warmContainer{
		contID:     contID,
		Expiration: expiration,
	})
}

func newFunctionPool(f *function.Function) *ContainerPool {
	fp := &ContainerPool{}
	fp.busy = list.New()
	fp.ready = list.New()

	return fp
}

// AcquireWarmContainer acquires a warm container for a given function (if any).
// A warm container is in running/paused state and has already been initialized
// with the function code.
// The acquired container is already in the busy pool.
// The function returns an error if either:
// (i) the warm container does not exist
// (ii) there are not enough resources to start the container
func AcquireWarmContainer(f *function.Function) (container.ContainerID, error) {
	Resources.Lock()
	defer Resources.Unlock()

	if Resources.AvailableCPUs < f.CPUDemand {
		log.Printf("Not enough CPU to start a warm container for %s", f)
		return "", OutOfResourcesErr
	}

	fp := getFunctionPool(f)
	contID, found := fp.acquireReadyContainer()
	if found {
		Resources.AvailableCPUs -= f.CPUDemand
		log.Printf("Acquired resources for warm container. Now: %v", Resources)
		return contID, nil
	}

	return "", NoWarmFoundErr
}

// ReleaseContainer puts a container in the ready pool for a function.
func ReleaseContainer(contID container.ContainerID, f *function.Function) {
	log.Printf("Container released for %v: %v", f, contID)

	Resources.Lock()
	defer Resources.Unlock()

	fp := getFunctionPool(f)

	// setup Expiration as time duration from now
	d := time.Duration(config.GetInt(config.CONTAINER_EXPIRATION_TIME, 600)) * time.Second
	fp.putReadyContainer(contID, time.Now().Add(d).UnixNano())

	// we must update the busy list by removing this element
	elem := fp.busy.Front()
	for ok := elem != nil; ok; ok = elem != nil {
		if elem.Value.(container.ContainerID) == contID {
			fp.busy.Remove(elem) // delete the element from the busy list
			break
		}
		elem = elem.Next()
	}

	Resources.AvailableCPUs += f.CPUDemand

	log.Printf("Released resources. Now: %v", Resources)
}

//NewContainer creates and starts a new container for the given function.
//The container can be directly used to schedule a request, as it is already
//in the busy pool.
func NewContainer(fun *function.Function) (container.ContainerID, error) {
	var image string
	if fun.Runtime == container.CUSTOM_RUNTIME {
		image = fun.CustomImage
	} else {
		runtime, ok := container.RuntimeToInfo[fun.Runtime]
		if !ok {
			return "", fmt.Errorf("Invalid runtime: %s", fun.Runtime)
		}
		image = runtime.Image
	}

	Resources.Lock()
	// check resources
	if Resources.AvailableMemMB < fun.MemoryMB {
		enoughMem, _ := dismissContainer(fun.MemoryMB)
		if !enoughMem {
			log.Printf("Not enough memory for the new container.")
			Resources.Unlock()
			return "", OutOfResourcesErr
		}
	}
	if Resources.AvailableCPUs < fun.CPUDemand {
		log.Printf("Not enough CPU for the new container.")
		Resources.Unlock()
		return "", OutOfResourcesErr
	}

	Resources.AvailableMemMB -= fun.MemoryMB
	Resources.AvailableCPUs -= fun.CPUDemand

	log.Printf("Acquired resources for new container. Now: %v", Resources)
	Resources.Unlock()

	contID, err := container.NewContainer(image, fun.TarFunctionCode, &container.ContainerOptions{
		MemoryMB: fun.MemoryMB,
	})

	Resources.Lock()
	defer Resources.Unlock()
	if err != nil {
		log.Printf("Failed container creation")
		Resources.AvailableMemMB += fun.MemoryMB
		Resources.AvailableCPUs += fun.CPUDemand
		return "", err
	}

	fp := getFunctionPool(fun)
	fp.putBusyContainer(contID) // We immediately mark it as busy

	return contID, nil
}

type itemToDismiss struct {
	contID container.ContainerID
	pool   *ContainerPool
	elem   *list.Element
	memory int64
}

// dismissContainer ... this function is used to get free memory used for a new container
// 2-phases: first, we find ready container and collect them as a slice, second (cleanup phase) we delete the container only and only if
// the sum of their memory is >= requiredMemoryMB is
func dismissContainer(requiredMemoryMB int64) (bool, error) {
	var cleanedMB int64 = 0
	var containerToDismiss []itemToDismiss
	res := false

	//first phase, research
	for _, funPool := range Resources.ContainerPools {
		if funPool.ready.Len() > 0 {
			// every container into the funPool has the same memory (same function)
			//so it is not important which one you destroy
			elem := funPool.ready.Front()
			contID := elem.Value.(warmContainer).contID
			// container in the same pool need same memory
			memory, _ := container.GetMemoryMB(contID)
			for ok := true; ok; ok = elem != nil {
				containerToDismiss = append(containerToDismiss,
					itemToDismiss{contID: contID, pool: funPool, elem: elem, memory: memory})
				cleanedMB += memory
				if cleanedMB >= requiredMemoryMB {
					goto cleanup
				}
				//go on to the next one
				elem = elem.Next()
			}
		}
	}

cleanup: // second phase, cleanup
	// memory check
	if cleanedMB >= requiredMemoryMB {
		for _, item := range containerToDismiss {
			item.pool.ready.Remove(item.elem)     // remove the container from the funPool
			err := container.Destroy(item.contID) // destroy the container
			if err != nil {
				res = false
				return res, nil
			}
			Resources.AvailableMemMB += item.memory
		}

		res = true
		log.Printf("Released resources. Now: %v", Resources)
	}
	return res, nil
}

// DeleteExpiredContainer is called by the container janitor
// Deletes expired warm container
func DeleteExpiredContainer() {
	now := time.Now().UnixNano()

	Resources.Lock()
	defer Resources.Unlock()

	for _, pool := range Resources.ContainerPools {
		elem := pool.ready.Front()
		for ok := elem != nil; ok; ok = elem != nil {
			warmed := elem.Value.(warmContainer)
			if now > warmed.Expiration {
				temp := elem
				elem = elem.Next()
				log.Printf("janitor: Removing container with ID %s\n", warmed.contID)
				pool.ready.Remove(temp) // remove the expired element

				memory, _ := container.GetMemoryMB(warmed.contID)
				container.Destroy(warmed.contID)
				Resources.AvailableMemMB += memory
				log.Printf("Released resources. Now: %v", Resources)

			} else {
				elem = elem.Next()
			}
		}
	}

}

// ShutdownAllContainers destroys all container (usually on termination)
func ShutdownAllContainers() {
	Resources.Lock()
	defer Resources.Unlock()

	for fun, pool := range Resources.ContainerPools {
		elem := pool.ready.Front()
		for ok := elem != nil; ok; ok = elem != nil {
			warmed := elem.Value.(warmContainer)
			temp := elem
			elem = elem.Next()
			log.Printf("Removing container with ID %s\n", warmed.contID)
			pool.ready.Remove(temp)

			memory, _ := container.GetMemoryMB(warmed.contID)
			container.Destroy(warmed.contID)
			Resources.AvailableMemMB += memory
		}

		function, _ := function.GetFunction(fun)

		elem = pool.busy.Front()
		for ok := elem != nil; ok; ok = elem != nil {
			contID := elem.Value.(container.ContainerID)
			temp := elem
			elem = elem.Next()
			log.Printf("Removing container with ID %s\n", contID)
			pool.ready.Remove(temp)

			memory, _ := container.GetMemoryMB(contID)
			container.Destroy(contID)
			Resources.AvailableMemMB += memory
			Resources.AvailableCPUs += function.CPUDemand
		}
	}
}

// WarmStatus foreach function returns the corresponding number of warm container available
func WarmStatus() (warmPool map[string]int) {
	Resources.RLock()
	defer Resources.RUnlock()
	warmPool = make(map[string]int)
	for funcName, pool := range Resources.ContainerPools {
		warmPool[funcName] = pool.ready.Len()
	}

	return warmPool
}
