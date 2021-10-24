package containers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/grussorusso/serverledge/internal/executor"
	"github.com/grussorusso/serverledge/internal/functions"
)

type ContainerID = string

func GetWarmContainer(f *functions.Function) (contID ContainerID, found bool) {
	found = false
	// TODO: check if we have a warm container for f
	// TODO: synchronization needed
	return contID, found
}

func WarmStart(r *functions.Request, c ContainerID) (string, error) {
	log.Printf("Starting warm container %v", c)
	return invoke(c, r)
}

func ColdStart(r *functions.Request) (string, error) {
	runtimeInfo := runtimeToInfo[r.Fun.Runtime]
	image := runtimeInfo.Image
	log.Printf("Starting new container for %s (image: %s)", r.Fun, image)

	// TODO: set memory

	contID, err := cf.Create(image, &ContainerOptions{})
	if err != nil {
		log.Printf("Failed container creation: %v", err)
		return "", err
	}

	content, ferr := os.Open(r.Fun.SourceTarURL) // TODO: HTTP
	defer content.Close()
	if ferr != nil {
		log.Fatalf("Reading failed: %v", ferr)
	}
	err = cf.CopyToContainer(contID, content, "/app/")
	if err != nil {
		log.Fatalf("Copy failed: %v", err)
	}

	err = cf.Start(contID)
	if err != nil {
		log.Fatalf("Starting container failed: %v", err)
	}

	return invoke(contID, r)
}

func invoke(contID string, r *functions.Request) (string, error) {
	cmd := runtimeToInfo[r.Fun.Runtime].InvocationCmd

	ipAddr, err := cf.GetIPAddress(contID)
	if err != nil {
		return "", fmt.Errorf("Failed to retrieve IP address for container: %v", err)
	}

	log.Printf("Invoking function on container: %v", ipAddr)

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
