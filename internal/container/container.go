package container

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/grussorusso/serverledge/internal/executor"
)

//NewContainer creates and starts a new container.
func NewContainer(image, codeTar string, opts *ContainerOptions) (ContainerID, error) {
	contID, err := cf.Create(image, opts)
	if err != nil {
		log.Printf("Failed container creation")
		return "", err
	}

	decodedCode, _ := base64.StdEncoding.DecodeString(codeTar)
	err = cf.CopyToContainer(contID, bytes.NewReader(decodedCode), "/app/")
	if err != nil {
		log.Printf("Failed code copy")
		return "", err
	}

	err = cf.Start(contID)
	if err != nil {
		return "", err
	}

	return contID, nil
}

// Execute interacts with the Executor running in the container to invoke the
// function through a HTTP request.
func Execute(contID ContainerID, req *executor.InvocationRequest) (*executor.InvocationResult, error) {
	ipAddr, err := cf.GetIPAddress(contID)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve IP address for container: %v", err)
	}

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

func GetMemoryMB(id ContainerID) (int64, error) {
	return cf.GetMemoryMB(id)
}

func Destroy(id ContainerID) error {
	return cf.Destroy(id)
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
