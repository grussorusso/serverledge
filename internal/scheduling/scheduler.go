package scheduling

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hexablock/vivaldi"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/logging"

	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/function"
)

var requests chan *scheduledRequest
var completions chan *scheduledRequest

func Run(p Policy) {
	requests = make(chan *scheduledRequest)
	completions = make(chan *scheduledRequest)

	// initialize Node resources
	availableCores := runtime.NumCPU()
	Node.AvailableMemMB = int64(config.GetInt(config.POOL_MEMORY_MB, 1024))
	Node.AvailableCPUs = config.GetFloat(config.POOL_CPUS, float64(availableCores)*2.0)
	Node.containerPools = make(map[string]*containerPool)
	Node.Coordinates = vivaldi.NewCoordinate(vivaldi.DefaultConfig())
	log.Printf("Current Node resources: %v", Node)

	container.InitDockerContainerFactory()

	//janitor periodically remove expired warm container
	GetJanitorInstance()

	// initialize scheduling policy
	p.Init()

	log.Println("Scheduler started.")

	// TODO: run policy asynchronously with "go ..."
	var r *scheduledRequest
	for {
		select {
		case r = <-requests:
			go p.OnArrival(r)
		case r = <-completions:
			go p.OnCompletion(r)
		}
	}

}

// SubmitRequest submits a newly arrived request for scheduling and execution
func SubmitRequest(r *function.Request) (*function.ExecutionReport, error) {
	log.Printf("New request for '%s' (class: %s, Max RespT: %f)", r.Fun, r.Class, r.MaxRespT)

	logger := logging.GetLogger()
	if !logger.Exists(r.Fun.Name) {
		logger.InsertNewLog(r.Fun.Name)
	}

	schedRequest := scheduledRequest{r, make(chan schedDecision, 1)}
	requests <- &schedRequest

	// wait on channel for scheduling action
	schedDecision, ok := <-schedRequest.decisionChannel
	if !ok {
		return nil, fmt.Errorf("Could not schedule the request!")
	}
	log.Printf("Sched action: %v", schedDecision)

	var report *function.ExecutionReport
	var err error
	if schedDecision.action == DROP {
		log.Printf("Dropping request")
		return nil, OutOfResourcesErr
	} else if schedDecision.action == EXEC_REMOTE {
		log.Printf("Offloading request")
		report, err = Offload(r, schedDecision.remoteHost)
		if err != nil {
			return nil, err
		}
	} else {
		report, err = Execute(schedDecision.contID, &schedRequest)
		if err != nil {
			return nil, err
		}
	}

	// TODO: SendReport also for dropped requests:
	// Pass "schedDecision" to SendReport to check for drops
	err = logger.SendReport(report, r.Fun.Name)
	if err != nil {
		log.Printf("unable to update log")
	}
	return report, nil
}

func handleColdStart(r *scheduledRequest, doOffload bool) {
	log.Printf("Cold start procedure for: %v", r)
	newContainer, err := newContainer(r.Fun)
	if errors.Is(err, OutOfResourcesErr) || err != nil {
		log.Printf("Could not create a new container: %v", err)
		if doOffload {
			handleOffload(r)
		} else {
			dropRequest(r)
		}
	} else {
		execLocally(r, newContainer, false)
	}
}

func dropRequest(r *scheduledRequest) {
	r.decisionChannel <- schedDecision{action: DROP}
}

func execLocally(r *scheduledRequest, c container.ContainerID, warmStart bool) {
	initTime := time.Now().Sub(r.Arrival).Seconds()
	r.Report = &function.ExecutionReport{InitTime: initTime, IsWarmStart: warmStart, Arrival: r.Arrival}

	decision := schedDecision{action: EXEC_LOCAL, contID: c}
	r.decisionChannel <- decision
}

func handleOffload(r *scheduledRequest) {
	r.decisionChannel <- schedDecision{
		action:     EXEC_REMOTE,
		contID:     "",
		remoteHost: config.GetString("server_url", "http://127.0.0.1:1324/invoke/"),
	}
}

func Offload(r *function.Request, serverUrl string) (*function.ExecutionReport, error) {
	// Prepare request
	request := function.InvocationRequest{Params: r.Params, QoSClass: r.Class, QoSMaxRespT: r.MaxRespT}
	invocationBody, err := json.Marshal(request)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	sendingTime := time.Now() // used to compute latency later on
	resp, err := http.Post(serverUrl+r.Fun.Name, "application/json",
		bytes.NewBuffer(invocationBody))

	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer resp.Body.Close()
	var report function.ExecutionReport
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &report)

	report.OffloadLatency = report.Arrival.Sub(sendingTime).Seconds()
	return &report, nil
}
