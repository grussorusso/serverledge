package scheduling

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/grussorusso/serverledge/internal/config"

	"github.com/grussorusso/serverledge/internal/containers"
	"github.com/grussorusso/serverledge/internal/functions"
)

var SchedRequests chan *functions.Request
var SchedCompletions chan *functions.Request

func Run() {
	SchedRequests = make(chan *functions.Request)
	SchedCompletions = make(chan *functions.ExecutionReport)

	log.Println("Scheduler started.")

	var r *functions.Request

	for {
		select {
		case r = <-SchedRequests:
			scheduleOnArrival(r)
		case <-SchedCompletions:
			// TODO: scheduleOnCompletion()
			log.Printf("Scheduler notified about completion.")
			return
		}
	}

}

func handleColdStart(r *functions.Request) {
	newContainer, err := containers.NewContainer(r.Fun)
	if errors.Is(err, containers.OutOfResourcesErr) {
		dropRequest(r)
	} else if err != nil {
		log.Printf("Could not create a new container: %v", err)
		dropRequest(r)
	} else {
		execLocally(r, newContainer)
	}
}

func dropRequest(r *functions.Request) {
	r.Sched <- SchedDecision{Decision: DROP}
}

func execLocally(r *functions.Request, c containers.ContainerID) {
	initTime := time.Now().Sub(r.Arrival).Seconds()
	r.Report = &functions.ExecutionReport{InitTime: initTime}

	decision := SchedDecision{Decision: EXEC_LOCAL, ContID: c}
	r.Sched <- decision
}

func scheduleOnArrival(r *functions.Request) {
	containerID, err := containers.AcquireWarmContainer(r.Fun)
	if err == nil {
		log.Printf("Using a warm container for: %v", r)
		execLocally(r, containerID)
	} else if errors.Is(err, containers.NoWarmFoundErr) {
		// Cold Start (handles asynchronously)
		go handleColdStart(r)
	} else if errors.Is(err, containers.OutOfResourcesErr) {
		log.Printf("Not enough resources on the node.")
		dropRequest(r)
	} else {
		// other error
		dropRequest(r)
	}
}

func Offload(r *functions.Request) (*http.Response, error) {
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
