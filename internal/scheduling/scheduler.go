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

	"github.com/grussorusso/serverledge/internal/client"
	"github.com/grussorusso/serverledge/internal/node"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/logging"

	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/function"
)

var requests chan *scheduledRequest
var completions chan *completion

var executionLogEnabled bool

var offloadingClient *http.Client

func Run(p Policy) {
	requests = make(chan *scheduledRequest, 500)
	completions = make(chan *completion, 500)

	// initialize Resources resources
	availableCores := runtime.NumCPU()
	node.Resources.AvailableMemMB = int64(config.GetInt(config.POOL_MEMORY_MB, 1024))
	node.Resources.AvailableCPUs = config.GetFloat(config.POOL_CPUS, float64(availableCores))
	node.Resources.ContainerPools = make(map[string]*node.ContainerPool)
	log.Printf("Current resources: %v", node.Resources)

	executionLogEnabled = config.GetBool(config.LOGGER_UPDATE_ENABLED, true)

	container.InitDockerContainerFactory()

	//janitor periodically remove expired warm container
	node.GetJanitorInstance()

	tr := &http.Transport{
		MaxIdleConns:        2500,
		MaxIdleConnsPerHost: 2500,
		MaxConnsPerHost:     0,
		IdleConnTimeout:     30 * time.Minute,
	}
	offloadingClient = &http.Client{Transport: tr}

	// initialize scheduling policy
	p.Init()

	log.Println("Scheduler started.")

	var r *scheduledRequest
	var c *completion
	for {
		select {
		case r = <-requests:
			go p.OnArrival(r)
		case c = <-completions:
			node.ReleaseContainer(c.contID, c.Fun)
			p.OnCompletion(r)
		}
	}

}

// SubmitRequest submits a newly arrived request for scheduling and execution
func SubmitRequest(r *function.Request) (*function.ExecutionReport, error) {
	var logger *logging.Logger = nil
	if executionLogEnabled {
		logger = logging.GetLogger()
		if !logger.Exists(r.Fun.Name) {
			logger.InsertNewLog(r.Fun.Name)
		}
	}

	remoteServerUrl := config.GetString(config.CLOUD_URL, "")

	schedRequest := scheduledRequest{
		Request:         r,
		decisionChannel: make(chan schedDecision, 1)}
	requests <- &schedRequest

	// wait on channel for scheduling action
	schedDecision, ok := <-schedRequest.decisionChannel
	if !ok {
		return nil, fmt.Errorf("could not schedule the request")
	}
	//log.Printf("[%s] Scheduling decision: %v", r, schedDecision)

	var report *function.ExecutionReport
	var err error
	if schedDecision.action == DROP {
		//log.Printf("[%s] Dropping request", r)
		return nil, node.OutOfResourcesErr
	} else if schedDecision.action == EXEC_REMOTE {
		//log.Printf("Offloading request")
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
	if executionLogEnabled && !(schedDecision.action == EXEC_REMOTE && schedDecision.remoteHost != remoteServerUrl) {
		err = logger.SendReport(report, r.Fun.Name)
		if err != nil {
			log.Printf("unable to update log")
		}
	}
	return report, nil
}

func handleColdStart(r *scheduledRequest) (isSuccess bool) {
	newContainer, err := node.NewContainer(r.Fun)
	if errors.Is(err, node.OutOfResourcesErr) || err != nil {
		log.Printf("Cold start failed: %v", err)
		return false
	} else {
		execLocally(r, newContainer, false)
		return true
	}
}

func dropRequest(r *scheduledRequest) {
	r.decisionChannel <- schedDecision{action: DROP}
}

func execLocally(r *scheduledRequest, c container.ContainerID, warmStart bool) {
	initTime := time.Now().Sub(r.Arrival).Seconds()
	r.Report = &function.ExecutionReport{InitTime: initTime, IsWarmStart: warmStart}

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
	request := client.InvocationRequest{Params: r.Params, QoSClass: int64(r.Class), QoSMaxRespT: r.MaxRespT}
	invocationBody, err := json.Marshal(request)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	sendingTime := time.Now() // used to compute latency later on
	resp, err := offloadingClient.Post(serverUrl+"/invoke/"+r.Fun.Name, "application/json",
		bytes.NewBuffer(invocationBody))

	if err != nil {
		log.Print(err)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Remote returned: %v", resp.StatusCode)
	}

	defer resp.Body.Close()
	var report function.ExecutionReport
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &report)

	// TODO: check how this is used in the QoSAware policy
	// It was originially computed as "report.Arrival - sendingTime"
	report.OffloadLatency = time.Now().Sub(sendingTime).Seconds() - report.Duration - report.InitTime
	report.SchedAction = "O"

	return &report, nil
}
