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

func Schedule(r *functions.Request) (*functions.ExecutionReport, error) {
	schedArrivalT := time.Now()
	containerID, err := containers.AcquireWarmContainer(r.Fun)
	if err == nil {
		log.Printf("Using a warm container for: %v", r)
	} else if errors.Is(err, containers.OutOfResourcesErr) {
		log.Printf("Not enough resources on the node.")
		return nil, err
	} else if errors.Is(err, containers.NoWarmFoundErr) {
		newContainer, err := containers.NewContainer(r.Fun)
		if errors.Is(err, containers.OutOfResourcesErr) {
			return nil, err
		} else if err != nil {
			return nil, fmt.Errorf("Could not create a new container: %v", err)
		}
		containerID = newContainer
	} else {
		return nil, err
	}

	initTime := time.Now().Sub(schedArrivalT).Seconds()
	r.Report = &functions.ExecutionReport{InitTime: initTime}

	return containers.Invoke(containerID, r)
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
