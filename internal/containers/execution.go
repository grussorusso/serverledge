package containers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/grussorusso/serverledge/internal/executor"
	"github.com/grussorusso/serverledge/internal/functions"
)

//Invoke serves a request on the specified container.
func Invoke(contID ContainerID, r *functions.Request) (string, error) {
	defer ReleaseContainer(contID, r.Fun)

	ipAddr, err := cf.GetIPAddress(contID)
	if err != nil {
		return "", fmt.Errorf("Failed to retrieve IP address for container: %v", err)
	}

	log.Printf("Invoking function on container: %v", ipAddr)

	cmd := runtimeToInfo[r.Fun.Runtime].InvocationCmd
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

// _invoke interacts with the Executor running in the container to invoke the
// function through a HTTP request.
func _invoke(ipAddr string, req *executor.InvocationRequest) (*executor.InvocationResult, error) {
	postBody, _ := json.Marshal(req)
	postBodyB := bytes.NewBuffer(postBody)
	resp, err := sendPostRequestWithRetries(fmt.Sprintf("http://%s:%d/invoke", ipAddr,
		executor.DEFAULT_EXECUTOR_PORT), postBodyB)
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

func sendPostRequestWithRetries(url string, body *bytes.Buffer) (*http.Response, error) {
	const maxRetries = 3
	const backoff = 300 * time.Millisecond

	var err error

	for retry := 1; retry <= maxRetries; retry++ {
		resp, err := http.Post(url, "application/json", body)
		if err == nil {
			return resp, err
		}

		time.Sleep(backoff)
	}

	return nil, err
}
