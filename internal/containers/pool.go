package containers

import (
	"bytes"
	"container/list"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/grussorusso/serverledge/internal/executor"
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

//Invoke serves a request on the specified container.
func Invoke(contID ContainerID, r *functions.Request) (string, error) {
	defer ReleaseContainer(contID, r.Fun)

	ipAddr, err := cf.GetIPAddress(contID)
	if err != nil {
		return "", fmt.Errorf("Failed to retrieve IP address for container: %v", err)
	}

	log.Printf("Invoking function on container: %v", ipAddr)

	cmd := runtimeToInfo[r.Fun.Runtime].InvocationCmd
	req := executor.InvocationRequest{
		cmd,
		r.Params,
		r.Fun.Handler,
		"/app",
	}
	response, err := _invoke(ipAddr, &req)
	if err != nil {
		return "", fmt.Errorf("Execution request failed: %v", err)
	}

	if !response.Success {
		return "", fmt.Errorf("Function execution failed")
	}

	return response.Result, nil
}

// _invoke interacts with the Executor running in the container to invoke the
// function through a HTTP request.
func _invoke(ipAddr string, req *executor.InvocationRequest) (*executor.InvocationResult, error) {
	postBody, _ := json.Marshal(req)
	postBodyB := bytes.NewBuffer(postBody)
	resp, err := http.Post(fmt.Sprintf("http://%s:%d/invoke", ipAddr,
		executor.DEFAULT_EXECUTOR_PORT), "application/json", postBodyB)
	if err != nil {
		return nil, fmt.Errorf("Request to executor failed: %v", err)
	}
	defer resp.Body.Close()

	d := json.NewDecoder(resp.Body)
	response := &executor.InvocationResult{}
	err = d.Decode(response)
	if err != nil {
		return nil, fmt.Errorf("Parsing executor response failed: %v", err)
	}

	return response, nil
}
