package containers

import (
	"bytes"
	"container/list"
	"encoding/base64"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/grussorusso/serverledge/internal/config"

	"github.com/grussorusso/serverledge/internal/functions"
)

type functionPool struct {
	sync.Mutex
	busy  *list.List
	ready *list.List //warm containers
}

type warmContainer struct {
	Expiration int64
	contID     ContainerID
}

var funToPool = make(map[string]*functionPool)
var functionPoolsMutex sync.Mutex

var usedMemoryMB int64 = 0 // MB
var memoryMutex sync.Mutex
var TotalMemoryMB int64

//getFunctionPool retrieves (or creates) the container pool for a function.
func getFunctionPool(f *functions.Function) *functionPool {
	functionPoolsMutex.Lock()
	defer functionPoolsMutex.Unlock()
	if fp, ok := funToPool[f.Name]; ok {
		return fp
	}

	fp := newFunctionPool(f)
	funToPool[f.Name] = fp
	return fp
}

func (fp *functionPool) acquireReadyContainer() (ContainerID, bool) {
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

func (fp *functionPool) putBusyContainer(contID ContainerID) {
	fp.busy.PushBack(contID)
}

func (fp *functionPool) putReadyContainer(contID ContainerID, expiration int64) {
	fp.ready.PushBack(warmContainer{
		contID:     contID,
		Expiration: expiration,
	})
}

func newFunctionPool(f *functions.Function) *functionPool {
	fp := &functionPool{}
	fp.busy = list.New()
	fp.ready = list.New()

	return fp
}

//AcquireWarmContainer acquires a warm container for a given function (if any).
//A warm container is in running/paused state and has already been initialized
//with the function code.
//The acquired container is already in the busy pool.
func AcquireWarmContainer(f *functions.Function) (contID ContainerID, found bool) {
	fp := getFunctionPool(f)
	fp.Lock()
	defer fp.Unlock()

	contID, found = fp.acquireReadyContainer()
	return
}

// ReleaseContainer puts a container in the ready pool for a function.
func ReleaseContainer(contID ContainerID, f *functions.Function) {
	//time.Sleep(15 * time.Second)
	log.Printf("Container released for %v: %v", f, contID)
	fp := getFunctionPool(f)
	fp.Lock()
	defer fp.Unlock()

	// setup Expiration as time duration from now
	//todo adjust default value
	d := time.Duration(config.GetInt("container.expiration", 30)) * time.Second
	fp.putReadyContainer(contID, time.Now().Add(d).UnixNano())

	// we must update the busy list by removing this element
	elem := fp.busy.Front()
	for ok := elem != nil; ok; ok = elem != nil {
		if elem.Value.(ContainerID) == contID {
			fp.busy.Remove(elem) // delete the element from the busy list
			break
		}
		elem.Next()
	}

}

//NewContainer creates and starts a new container for the given function.
//The container can be directly used to schedule a request, as it is already
//in the busy pool.
func NewContainer(fun *functions.Function) (ContainerID, error) {
	image := runtimeToInfo[fun.Runtime].Image

	memoryMutex.Lock()
	//memory check
	if TotalMemoryMB-usedMemoryMB < fun.MemoryMB {
		enoughMem, _ := dismissContainer(fun.MemoryMB)
		if !enoughMem {
			memoryMutex.Unlock()
			return "", errors.New("unable to create container: memory not available")
		}
	}

	usedMemoryMB += fun.MemoryMB
	memoryMutex.Unlock()
	log.Printf("Starting new container for %s (image: %s)", fun, image)
	contID, err := cf.Create(image, &ContainerOptions{
		MemoryMB: fun.MemoryMB,
	})
	if err != nil {
		return "", err
	}

	decodedCode, _ := base64.StdEncoding.DecodeString(fun.TarFunctionCode)
	err = cf.CopyToContainer(contID, bytes.NewReader(decodedCode), "/app/")
	if err != nil {
		return "", err
	}

	err = cf.Start(contID)
	if err != nil {
		return "", err
	}

	fp := getFunctionPool(fun)
	fp.Lock()
	defer fp.Unlock()
	fp.putBusyContainer(contID) // We immediately mark it as busy

	return contID, nil
}

type itemToDismiss struct {
	contID ContainerID
	pool   *functionPool
	elem   *list.Element
	memory int64
}

// dismissContainer ... this function is used to get free memory used for a new container
// 2-phases: first, we find ready containers and collect them as a slice, second (cleanup phase) we delete the containers only and only if
// the sum of their memory is >= requiredMemoryMB is
func dismissContainer(requiredMemoryMB int64) (bool, error) {
	functionPoolsMutex.Lock()
	defer functionPoolsMutex.Unlock()

	var cleanedMB int64 = 0
	var containerToDismiss []itemToDismiss
	var toUnlock []*functionPool
	res := false

	//first phase, research
	for _, funPool := range funToPool {
		funPool.Lock()
		if funPool.ready.Len() > 0 {
			toUnlock = append(toUnlock, funPool)
			// every container into the funPool has the same memory (same function)
			//so it is not important which one you destroy
			elem := funPool.ready.Front()
			contID := elem.Value.(warmContainer).contID
			// containers in the same pool need same memory
			memory, _ := cf.GetMemoryMB(contID)
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
		} else {
			// ready list is empty
			funPool.Unlock()
		}
	}

cleanup: // second phase, cleanup
	// memory check
	if cleanedMB >= requiredMemoryMB {
		for _, item := range containerToDismiss {
			item.pool.ready.Remove(item.elem) // remove the container from the funPool
			err := cf.Destroy(item.contID)    // destroy the container
			if err != nil {
				res = false
				goto unlock
			}
			usedMemoryMB -= item.memory
		}

		res = true
	}

unlock:
	for _, elem := range toUnlock {
		elem.Unlock()
	}

	return res, nil
}

// DeleteExpiredContainer is called by the container janitor
// Deletes expired warm containers
func DeleteExpiredContainer() {
	now := time.Now().UnixNano()

	functionPoolsMutex.Lock()
	defer functionPoolsMutex.Unlock()

	for _, pool := range funToPool {
		pool.Lock()

		elem := pool.ready.Front()
		for ok := elem != nil; ok; ok = elem != nil {
			warmed := elem.Value.(warmContainer)
			if now > warmed.Expiration {
				temp := elem
				elem = elem.Next()
				log.Printf("janitor: Removing container with ID %s\n", warmed.contID)
				pool.ready.Remove(temp) // remove the expired element

				memoryMutex.Lock()
				memory, _ := cf.GetMemoryMB(warmed.contID)
				cf.Destroy(warmed.contID)
				usedMemoryMB -= memory
				memoryMutex.Unlock()

			} else {
				elem = elem.Next()
			}
		}

		pool.Unlock()
	}

}
