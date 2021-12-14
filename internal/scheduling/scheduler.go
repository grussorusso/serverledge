package scheduling

import (
	"bytes"
	"encoding/json"
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
	containerID, ok := containers.AcquireWarmContainer(r.Fun)
	if !ok {
		newContainer, err := containers.NewContainer(r.Fun)
		if err != nil {
			// TODO: this may fail because we run out of memory/CPU
			// handle this error differently?
			return nil, fmt.Errorf("Could not create a new container: %v", err)
		}
		containerID = newContainer
	} else {
		log.Printf("Using a warm container for: %v", r)
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
