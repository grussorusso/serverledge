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

var requests chan *ScheduledRequest
var completions chan *ScheduledRequest

func Run() {
	requests = make(chan *ScheduledRequest)
	completions = make(chan *ScheduledRequest)

	// initialize node resources
	availableCores := runtime.NumCPU()
	node.AvailableMemMB = int64(config.GetInt(config.POOL_MEMORY_MB, 1024))
	node.AvailableCPUs = config.GetFloat(config.POOL_CPUS, float64(availableCores)*2.0)
	node.containerPools = make(map[string]*containerPool)
	log.Printf("Current node resources: %v", node)

	container.InitDockerContainerFactory()

	//janitor periodically remove expired warm container
	GetJanitorInstance()

	log.Println("Scheduler started.")

	var r *ScheduledRequest

	for {
		select {
		case r = <-requests:
			log.Printf("Scheduler notified about arrival.")
			scheduleOnArrival(r)
		case <-completions:
			// TODO: scheduleOnCompletion()
			log.Printf("Scheduler notified about completion.")
		}
	}

}

// SubmitRequest submits a newly arrived request for scheduling and execution
func SubmitRequest(r *function.Request) (*function.ExecutionReport, error) {
	log.Printf("New request for '%s' (class: %s, Max RespT: %f)", r.Fun, r.Class, r.MaxRespT)

	schedRequest := ScheduledRequest{r, make(chan SchedDecision, 1)}
	requests <- &schedRequest

	// wait on channel for scheduling decision
	schedDecision, ok := <-schedRequest.decisionChannel
	if !ok {
		return nil, fmt.Errorf("Could not schedule the request!")
	}
	log.Printf("Sched decision: %v", schedDecision)

	if schedDecision.Decision == DROP {
		log.Printf("Dropping request")
		return nil, OutOfResourcesErr
	} else {
		result, err := Execute(schedDecision.ContID, &schedRequest)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
}

func handleColdStart(r *ScheduledRequest) {
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

func dropRequest(r *ScheduledRequest) {
	r.decisionChannel <- SchedDecision{Decision: DROP}
}

func execLocally(r *ScheduledRequest, c container.ContainerID) {
	initTime := time.Now().Sub(r.Arrival).Seconds()
	r.Report = &function.ExecutionReport{InitTime: initTime}

	decision := SchedDecision{Decision: EXEC_LOCAL, ContID: c}
	r.decisionChannel <- decision
}

func scheduleOnArrival(r *ScheduledRequest) {
	containerID, err := acquireWarmContainer(r.Fun)
	if err == nil {
		log.Printf("Using a warm container for: %v", r)
		execLocally(r, containerID)
	} else if errors.Is(err, NoWarmFoundErr) {
		// Cold Start (handles asynchronously)
		go handleColdStart(r)
	} else if errors.Is(err, OutOfResourcesErr) {
		log.Printf("Not enough resources on the node.")
		dropRequest(r)
	} else {
		// other error
		dropRequest(r)
	}
}

func Offload(r *ScheduledRequest) (*http.Response, error) {
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
