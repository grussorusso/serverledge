package containers

import (
	"container/list"
	"log"
	"os"
	"sync"

	"github.com/grussorusso/serverledge/internal/functions"
)

type functionPool struct {
	sync.Mutex
	busy  *list.List
	ready *list.List
}

var funToPool map[string]*functionPool = make(map[string]*functionPool)
var functionPoolsMutex sync.Mutex

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
	contID := elem.Value.(ContainerID)
	fp.putBusyContainer(contID)

	return contID, true
}

func (fp *functionPool) putBusyContainer(contID ContainerID) {
	fp.busy.PushBack(contID)
}

func (fp *functionPool) putReadyContainer(contID ContainerID) {
	fp.ready.PushBack(contID)
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
	log.Printf("Container released for %v: %v", f, contID)
	fp := getFunctionPool(f)
	fp.Lock()
	defer fp.Unlock()

	fp.putReadyContainer(contID)
}

//NewContainer creates and starts a new container for the given function.
//The container can be directly used to schedule a request, as it is already
//in the busy pool.
func NewContainer(fun *functions.Function) (ContainerID, error) {
	image := runtimeToInfo[fun.Runtime].Image
	log.Printf("Starting new container for %s (image: %s)", fun, image)

	// TODO: set memory

	// TODO: acquire resources with synchronization

	contID, err := cf.Create(image, &ContainerOptions{})
	if err != nil {
		return "", err
	}

	content, ferr := os.Open(fun.SourceTarURL) // TODO: HTTP
	defer content.Close()
	if ferr != nil {
		return "", ferr
	}
	err = cf.CopyToContainer(contID, content, "/app/")
	if err != nil {
		return "", ferr
	}

	err = cf.Start(contID)
	if err != nil {
		return "", ferr
	}

	fp := getFunctionPool(fun)
	fp.Lock()
	defer fp.Unlock()
	fp.putBusyContainer(contID) // We immediately mark it as busy

	return contID, nil
}
