package scheduling

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/grussorusso/serverledge/internal/config"

	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/function"
)

var requests chan *scheduledRequest
var completions chan *scheduledRequest

func Run(p Policy) {
	requests = make(chan *scheduledRequest)
	completions = make(chan *scheduledRequest)

	// initialize node resources
	availableCores := runtime.NumCPU()
	node.AvailableMemMB = int64(config.GetInt(config.POOL_MEMORY_MB, 1024))
	node.AvailableCPUs = config.GetFloat(config.POOL_CPUS, float64(availableCores)*2.0)
	node.containerPools = make(map[string]*containerPool)
	log.Printf("Current node resources: %v", node)

	container.InitDockerContainerFactory()

	//janitor periodically remove expired warm container
	GetJanitorInstance()

	// initialize scheduling policy
	p.Init()

	log.Println("Scheduler started.")

	var r *scheduledRequest
	for {
		select {
		case r = <-requests:
			p.OnArrival(r)
		case r = <-completions:
			p.OnCompletion(r)
		}
	}

}

// SubmitRequest submits a newly arrived request for scheduling and execution
func SubmitRequest(r *function.Request) (*function.ExecutionReport, error) {
	log.Printf("New request for '%s' (class: %s, Max RespT: %f)", r.Fun, r.Class, r.MaxRespT)

	schedRequest := scheduledRequest{r, make(chan schedDecision, 1)}
	requests <- &schedRequest

	// wait on channel for scheduling action
	schedDecision, ok := <-schedRequest.decisionChannel
	if !ok {
		return nil, fmt.Errorf("Could not schedule the request!")
	}
	log.Printf("Sched action: %v", schedDecision)

	if schedDecision.action == DROP {
		log.Printf("Dropping request")
		return nil, OutOfResourcesErr
	} else {
		result, err := Execute(schedDecision.contID, &schedRequest)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
}

func handleColdStart(r *scheduledRequest) {
	newContainer, err := newContainer(r.Fun)
	if errors.Is(err, OutOfResourcesErr) {
		dropRequest(r)
	} else if err != nil {
		log.Printf("Could not create a new container: %v", err)
		dropRequest(r)
	} else {
		execLocally(r, newContainer)
	}
}

func dropRequest(r *scheduledRequest) {
	r.decisionChannel <- schedDecision{action: DROP}
}

func execLocally(r *scheduledRequest, c container.ContainerID) {
	initTime := time.Now().Sub(r.Arrival).Seconds()
	r.Report = &function.ExecutionReport{InitTime: initTime}

	decision := schedDecision{action: EXEC_LOCAL, contID: c}
	r.decisionChannel <- decision
}

func Offload(r *scheduledRequest) (*http.Response, error) {
	serverUrl := config.GetString("server_url", "http://127.0.0.1:1324/invoke/")
	jsonData, err := json.Marshal(r.Params)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	resp, err := http.Post(serverUrl+r.Fun.Name, "application/json",
		bytes.NewBuffer(jsonData))

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return resp, nil
}
