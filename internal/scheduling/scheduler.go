package scheduling

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/grussorusso/serverledge/internal/node"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/logging"

	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/function"
)

var requests chan *scheduledRequest
var completions chan *scheduledRequest

func Run(p Policy) {
	requests = make(chan *scheduledRequest, 500)
	completions = make(chan *scheduledRequest, 500)

	// initialize Resources resources
	availableCores := runtime.NumCPU()
	node.Resources.AvailableMemMB = int64(config.GetInt(config.POOL_MEMORY_MB, 1024))
	node.Resources.AvailableCPUs = config.GetFloat(config.POOL_CPUS, float64(availableCores))
	node.Resources.ContainerPools = make(map[string]*node.ContainerPool)
	log.Printf("Current Resources resources: %v", node.Resources)

	container.InitDockerContainerFactory()

	//janitor periodically remove expired warm container
	node.GetJanitorInstance()

	// initialize scheduling policy
	p.Init()

	log.Println("Scheduler started.")

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
	select { // non-blocking send,if decision is not taken in time drop the request
	case requests <- &schedRequest:
		break
	case <-time.After(time.Duration(r.RequestQoS.MaxRespT) * time.Second):
		schedRequest.decisionChannel <- schedDecision{action: DROP}
		break
	}

	// wait on channel for scheduling action
	schedDecision, ok := <-schedRequest.decisionChannel
	if !ok {
		return nil, fmt.Errorf("could not schedule the request")
	}
	log.Printf("Sched action: %v", schedDecision)

	var report *function.ExecutionReport
	var err error
	if schedDecision.action == DROP {
		log.Printf("Dropping request")
		return nil, node.OutOfResourcesErr
	} else if schedDecision.action == EXEC_REMOTE {
		log.Printf("CanDoOffloading request")
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
	if !(schedDecision.action == EXEC_REMOTE && schedDecision.remoteHost != remoteServerUrl) {
		err = logger.SendReport(report, r.Fun.Name)
		if err != nil {
			log.Printf("unable to update log")
		}
	}
	return report, nil
}

func handleColdStart(r *scheduledRequest) (isSuccess bool) {
	log.Printf("Cold start procedure for: %v", r)
	newContainer, err := node.NewContainer(r.Fun)
	if errors.Is(err, node.OutOfResourcesErr) || err != nil {
		log.Printf("Could not create a new container: %v", err)
		return false
	} else {
		execLocally(r, newContainer, false)
		return true
	}
}

func dropRequest(r *scheduledRequest) {
	if dropManager != nil {
		dropManager.sendDropAlert() // TODO: this is policy-specific
	}
	r.decisionChannel <- schedDecision{action: DROP}
}

func execLocally(r *scheduledRequest, c container.ContainerID, warmStart bool) {
	initTime := time.Now().Sub(r.Arrival).Seconds()
	r.Report = &function.ExecutionReport{InitTime: initTime, IsWarmStart: warmStart, Arrival: r.Arrival}

	decision := schedDecision{action: EXEC_LOCAL, contID: c}
	r.decisionChannel <- decision
}

func handleOffload(r *scheduledRequest, serverHost string) {
	r.CanDoOffloading = false // the next server can't offload this request
	r.decisionChannel <- schedDecision{
		action:     EXEC_REMOTE,
		contID:     "",
		remoteHost: serverHost,
	}
}

func handleCloudOffload(r *scheduledRequest) {
	cloudAddress := config.GetString(config.CLOUD_URL, "")
	handleOffload(r, cloudAddress)
}

func Offload(r *function.Request, serverUrl string) (*function.ExecutionReport, error) {
	// Prepare request
	request := function.InvocationRequest{Params: r.Params, QoSClass: r.Class, QoSMaxRespT: r.MaxRespT}
	invocationBody, err := json.Marshal(request)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	sendingTime := time.Now() // used to compute latency later on
	resp, err := http.Post(serverUrl+"/invoke/"+r.Fun.Name, "application/json",
		bytes.NewBuffer(invocationBody))

	if err != nil {
		log.Print(err)
		return nil, err
	}
	defer resp.Body.Close()
	var report function.ExecutionReport
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &report)

	report.OffloadLatency = report.Arrival.Sub(sendingTime).Seconds()
	return &report, nil
}
