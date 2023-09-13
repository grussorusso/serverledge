package test

import (
	"encoding/json"
	"fmt"
	"github.com/grussorusso/serverledge/internal/client"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/utils"
	"testing"
	"time"
)

func TestContainerPool(t *testing.T) {
	if !INTEGRATION_TEST {
		t.FailNow()
	}
	funcs := []string{"inc", "double"}
	for _, name := range funcs {
		fn, err := initializePyFunction(name, "handler", function.NewSignature().
			AddInput("input", function.Int{}).
			AddOutput("result", function.Int{}).
			Build())
		utils.AssertNil(t, err)

		createTest(t, fn)
	}

	channel := make(chan error)
	const n = 3
	for i := 0; i < n; i++ {
		for _, name := range funcs {
			x := make(map[string]interface{})
			x["input"] = 1
			fnName := name
			go func() {
				time.Sleep(50 * time.Millisecond)
				err := invokeTest(fnName, x)
				channel <- err
			}()
		}
	}

	// wait for all functions to complete and checking the errors
	for i := 0; i < len(funcs)*n; i++ {
		err := <-channel
		utils.AssertNil(t, err)
	}

	for _, name := range funcs {
		deleteTest(t, name)
	}

}

func createTest(t *testing.T, fn *function.Function) {
	marshal, err := json.Marshal(fn)
	utils.AssertNil(t, err)

	postJson, err := utils.PostJson("http://localhost:1323/create", marshal)
	utils.AssertNil(t, err)

	utils.PrintJsonResponse(postJson.Body)
}

func invokeTest(fn string, params map[string]interface{}) error {
	request := client.InvocationRequest{
		Params: params,
		// QoSClass:        qosClass,
		QoSMaxRespT:     250,
		CanDoOffloading: true,
		Async:           false,
	}
	invocationBody, err1 := json.Marshal(request)
	if err1 != nil {
		return err1
	}
	url := fmt.Sprintf("http://localhost:1323/invoke/%s", fn)
	_, err2 := utils.PostJson(url, invocationBody)
	if err2 != nil {
		return err2
	}
	// utils.PrintJsonResponse(resp.Body)
	return nil
}

func deleteTest(t *testing.T, fn string) {
	request := function.Function{Name: fn}
	requestBody, err := json.Marshal(request)
	utils.AssertNil(t, err)

	url := fmt.Sprintf("http://localhost:1323/delete")
	resp, err := utils.PostJson(url, requestBody)
	utils.AssertNil(t, err)

	utils.PrintJsonResponse(resp.Body)
}
