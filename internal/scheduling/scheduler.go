package scheduling

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/grussorusso/serverledge/internal/config"

	"github.com/grussorusso/serverledge/internal/containers"
	"github.com/grussorusso/serverledge/internal/functions"
)

var SchedRequests chan *functions.Request
var SchedCompletions chan *functions.ExecutionReport

func Run() {
	SchedRequests = make(chan *functions.Request)
	SchedCompletions = make(chan *functions.ExecutionReport)

	log.Println("Scheduler started.")

	var r *functions.Request

	for {
		select {
		case r = <-SchedRequests:
			handleRequest(r)
		case <-SchedCompletions:
			fmt.Println("completion")
			return
		}
	}

}

func handleRequest(r *functions.Request) {
	schedArrivalT := time.Now()
	containerID, err := containers.AcquireWarmContainer(r.Fun)
	if err == nil {
		log.Printf("Using a warm container for: %v", r)
	} else if errors.Is(err, containers.OutOfResourcesErr) {
		log.Printf("Not enough resources on the node.")
		r.Sched <- SchedDecision{Decision: DROP}
		return
	} else if errors.Is(err, containers.NoWarmFoundErr) {
		newContainer, err := containers.NewContainer(r.Fun)
		if errors.Is(err, containers.OutOfResourcesErr) {
			r.Sched <- SchedDecision{Decision: DROP}
			return
		} else if err != nil {
			log.Printf("Could not create a new container: %v", err)
			r.Sched <- SchedDecision{Decision: DROP}
			return
		}
		containerID = newContainer
	} else {
		r.Sched <- SchedDecision{Decision: DROP}
		return
	}

	initTime := time.Now().Sub(schedArrivalT).Seconds()
	r.Report = &functions.ExecutionReport{InitTime: initTime}

	// return decision
	decision := SchedDecision{Decision: EXEC_LOCAL, ContID: containerID}
	r.Sched <- decision
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
