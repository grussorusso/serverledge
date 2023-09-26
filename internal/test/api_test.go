package test

import (
	"encoding/json"
	"fmt"
	"github.com/grussorusso/serverledge/internal/client"
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/utils"
	"net/http"
	"testing"
	"time"
)

// TestContainerPool executes repeatedly different functions (**not compositions**) to verify the container pool
func TestContainerPool(t *testing.T) {
	if !INTEGRATION_TEST {
		t.Skip()
	}
	// creating inc and double functions
	funcs := []string{"inc", "double"}
	for _, name := range funcs {
		fn, err := initializePyFunction(name, "handler", function.NewSignature().
			AddInput("input", function.Int{}).
			AddOutput("result", function.Int{}).
			Build())
		utils.AssertNil(t, err)

		createTest(t, fn, HOST, PORT)
	}
	// getting functions
	functionNames := getFunctionTest(t, HOST, PORT)
	utils.AssertSliceEquals(t, []string{"double", "inc"}, functionNames)
	// executing all functions
	channel := make(chan error)
	const n = 3
	for i := 0; i < n; i++ {
		for _, name := range funcs {
			x := make(map[string]interface{})
			x["input"] = 1
			fnName := name
			go func() {
				time.Sleep(50 * time.Millisecond)
				err := invokeTest(fnName, x, HOST, PORT)
				channel <- err
			}()
		}
	}

	// wait for all functions to complete and checking the errors
	for i := 0; i < len(funcs)*n; i++ {
		err := <-channel
		utils.AssertNil(t, err)
	}
	// delete each function
	for _, name := range funcs {
		deleteTest(t, name, HOST, PORT)
	}

}

// TestCreateComposition tests the compose REST API that creates a new function composition
func TestCreateComposition(t *testing.T) {
	fcName := "sequence"
	fn, err := initializePyFunction("inc", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())
	utils.AssertNilMsg(t, err, "failed to initialize function")
	dag, err := fc.CreateSequenceDag(fn, fn, fn)
	composition := fc.NewFC(fcName, *dag, []*function.Function{fn}, true)
	createCompositionTest(t, &composition, HOST, PORT)

	// verifies the function exists (using function REST API)
	functionNames := getFunctionTest(t, HOST, PORT)
	utils.AssertSliceEquals(t, []string{"inc"}, functionNames)

	// here we do not use REST API
	getFC, b := fc.GetFC(fcName)
	utils.AssertTrue(t, b)
	utils.AssertTrueMsg(t, composition.Equals(getFC), "composition comparison failed")
	err = composition.Delete()
	utils.AssertNilMsg(t, err, "failed to delete composition")

	// verifies the function does not exists  (using function REST API)
	functionNames = getFunctionTest(t, HOST, PORT)
	utils.AssertSliceEquals(t, []string{}, functionNames)
}

func createTest(t *testing.T, fn *function.Function, host string, port int) {
	marshaledFunc, err := json.Marshal(fn)
	utils.AssertNil(t, err)
	url := fmt.Sprintf("http://%s:%d/create", host, port)
	postJson, err := utils.PostJson(url, marshaledFunc)
	utils.AssertNil(t, err)

	utils.PrintJsonResponse(postJson.Body)
}

func invokeTest(fn string, params map[string]interface{}, host string, port int) error {
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
	url := fmt.Sprintf("http://%s:%d/invoke/%s", host, port, fn)
	_, err2 := utils.PostJson(url, invocationBody)
	if err2 != nil {
		return err2
	}
	// utils.PrintJsonResponse(resp.Body)
	return nil
}

func getFunctionTest(t *testing.T, host string, port int) []string {
	url := fmt.Sprintf("http://%s:%d/function", host, port)
	resp, err := http.Get(url)
	utils.AssertNil(t, err)
	var functionNames []string
	functionListJson := utils.GetJsonResponse(resp.Body)
	err = json.Unmarshal([]byte(functionListJson), &functionNames)
	utils.AssertNil(t, err)
	return functionNames
}

func deleteTest(t *testing.T, fn string, host string, port int) {
	request := function.Function{Name: fn}
	requestBody, err := json.Marshal(request)
	utils.AssertNil(t, err)

	url := fmt.Sprintf("http://%s:%d/delete", host, port)
	resp, err := utils.PostJson(url, requestBody)
	utils.AssertNil(t, err)

	utils.PrintJsonResponse(resp.Body)
}

func createCompositionTest(t *testing.T, fc *fc.FunctionComposition, host string, port int) {
	marshaledFunc, err := json.Marshal(fc)
	utils.AssertNilMsg(t, err, "failed to marshal composition")
	url := fmt.Sprintf("http://%s:%d/compose", host, port)
	postJson, err := utils.PostJson(url, marshaledFunc)
	utils.AssertNilMsg(t, err, "failed to create composition")

	utils.PrintJsonResponse(postJson.Body)
}

func invokeCompositionTest(t *testing.T, params map[string]interface{}, fc string, host string, port int) {
	request := client.InvocationRequest{
		Params: params,
		// QoSClass:        qosClass,
		QoSMaxRespT:     250,
		CanDoOffloading: true,
		Async:           false,
	}
	invocationBody, err1 := json.Marshal(request)
	utils.AssertNilMsg(t, err1, "error while marshaling invocation request for composition")

	url := fmt.Sprintf("http://%s:%d/play/%s", host, port, fc)
	resp, err2 := utils.PostJson(url, invocationBody)
	utils.AssertNilMsg(t, err2, "error while posting json request for invoking a composition")
	utils.PrintJsonResponse(resp.Body)
}

func getCompositionsTest(t *testing.T, host string, port int) []string {
	url := fmt.Sprintf("http://%s:%d/fc", host, port)
	resp, err := http.Get(url)
	utils.AssertNil(t, err)
	var fcNames []string
	functionListJson := utils.GetJsonResponse(resp.Body)
	err = json.Unmarshal([]byte(functionListJson), &fcNames)
	utils.AssertNil(t, err)
	return fcNames
}

func deleteCompositionTest(t *testing.T, fcName string, host string, port int) {
	request := fc.FunctionComposition{Name: fcName}
	requestBody, err := json.Marshal(request)
	utils.AssertNil(t, err)

	url := fmt.Sprintf("http://%s:%d/uncompose", host, port)
	resp, err := utils.PostJson(url, requestBody)
	utils.AssertNil(t, err)

	utils.PrintJsonResponse(resp.Body)
}
