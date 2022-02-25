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

var chHA = make(chan *function.ExecutionReport)

func TestHAServiceClient(t *testing.T) {
	go invokeHAFunction(t, 10)
	result := <-chHA
	t.Logf("%v", result)

	go invokeHAFunction(t, 5)
	result = <-chHA
	t.Logf("%v", result)

	go invokeHAFunction(t, 1)
	go invokeHAFunction(t, 0.7)
	res1 := <-chHA
	res2 := <-chHA
	t.Logf("%v", res1)
	t.Logf("%v", res2)
	if res2.OffloadLatency != 0 || res1.OffloadLatency != 0 {
		t.Log("stop test")
	}

}

func invokeHAFunction(t *testing.T, respT float64) {
	params := make(map[string]string)
	params["a"] = "a"
	params["b"] = "b"
	// Prepare request
	request := function.InvocationRequest{Params: params, QoSClass: function.HIGH_AVAILABILITY, QoSMaxRespT: respT, Offloading: true}
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
	chHA <- &result
}
