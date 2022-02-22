package test

import (
	"encoding/json"
	"fmt"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/utils"
	"io/ioutil"
	"os"
	"testing"
)

var ch = make(chan *function.ExecutionReport)

func TestLowServiceClient(t *testing.T) {
	go invokeFunction(t)
	result := <-ch
	t.Log(result)

	go invokeFunction(t)
	result = <-ch
	t.Log(result)

	go invokeFunction(t)
	go invokeFunction(t)
	res1 := <-ch
	res2 := <-ch
	t.Log(res1)
	t.Log(res2)
	if res2.OffloadLatency != 0 || res1.OffloadLatency != 0 {
		t.Log("stop test")
	}

}

func invokeFunction(t *testing.T) {
	params := make(map[string]string)
	params["a"] = "a"
	params["b"] = "b"
	// Prepare request
	request := function.InvocationRequest{Params: params, QoSClass: function.LOW, QoSMaxRespT: 2, Offloading: true}
	invocationBody, err := json.Marshal(request)
	if err != nil {
		t.Log(err)
		return
	}
	t.Logf("request %v", invocationBody)
	// Send invocation request
	url := fmt.Sprintf("http://%s:%d/invoke/%s", "127.0.0.1", 1323, "func")
	resp, err := utils.PostJson(url, invocationBody)
	if err != nil {
		fmt.Printf("Invocation failed: %v", err)
		os.Exit(1)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body) // response body is []byte
	if err != nil {
		fmt.Printf("ReadAll failed: %v", err)
		os.Exit(1)
	}

	var result function.ExecutionReport
	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println("Can not unmarshal JSON")
		os.Exit(1)
	}
	ch <- &result
}
