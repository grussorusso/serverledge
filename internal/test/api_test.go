package test

import (
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/utils"
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

		createApiTest(t, fn, HOST, PORT)
	}
	// getting functions
	functionNames := getFunctionApiTest(t, HOST, PORT)
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
				err := invokeApiTest(fnName, x, HOST, PORT)
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
		deleteApiTest(t, name, HOST, PORT)
	}

}

// TestCreateComposition tests the compose REST API that creates a new function composition
func TestCreateComposition(t *testing.T) {
	if !INTEGRATION_TEST {
		t.Skip()
	}
	fcName := "sequence"
	fn, err := initializePyFunction("inc", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())
	utils.AssertNilMsg(t, err, "failed to initialize function")
	dag, err := fc.CreateSequenceDag(fn, fn, fn)
	composition := fc.NewFC(fcName, *dag, []*function.Function{fn}, true)
	createCompositionApiTest(t, &composition, HOST, PORT)

	// verifies the function exists (using function REST API)
	functionNames := getFunctionApiTest(t, HOST, PORT)
	utils.AssertSliceEquals(t, []string{"inc"}, functionNames)

	// here we do not use REST API
	getFC, b := fc.GetFC(fcName)
	utils.AssertTrue(t, b)
	utils.AssertTrueMsg(t, composition.Equals(getFC), "composition comparison failed")
	err = composition.Delete()
	utils.AssertNilMsg(t, err, "failed to delete composition")

	// verifies the function does not exists  (using function REST API)
	functionNames = getFunctionApiTest(t, HOST, PORT)
	utils.AssertSliceEquals(t, []string{}, functionNames)
}

// TestInvokeComposition tests the REST API that executes a given function composition
func TestInvokeComposition(t *testing.T) {
	if !INTEGRATION_TEST {
		t.Skip()
	}
	fcName := "sequence"
	fn, err := initializePyFunction("inc", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())
	utils.AssertNilMsg(t, err, "failed to initialize function")
	dag, err := fc.CreateSequenceDag(fn, fn, fn)
	composition := fc.NewFC(fcName, *dag, []*function.Function{fn}, true)
	createCompositionApiTest(t, &composition, HOST, PORT)

	// verifies the function exists (using function REST API)
	functionNames := getFunctionApiTest(t, HOST, PORT)
	utils.AssertSliceEquals(t, []string{"inc"}, functionNames)

	// === this is the test ===
	params := make(map[string]interface{})
	params["input"] = 1
	invokeCompositionApiTest(t, params, fcName, HOST, PORT, false)

	// here we do not use REST API
	getFC, b := fc.GetFC(fcName)
	utils.AssertTrue(t, b)
	utils.AssertTrueMsg(t, composition.Equals(getFC), "composition comparison failed")
	err = composition.Delete()
	utils.AssertNilMsg(t, err, "failed to delete composition")

	// verifies the function does not exists  (using function REST API)
	functionNames = getFunctionApiTest(t, HOST, PORT)
	utils.AssertSliceEquals(t, []string{}, functionNames)
}

// TestDeleteComposition tests the compose REST API that deletes a function composition
func TestDeleteComposition(t *testing.T) {
	if !INTEGRATION_TEST {
		t.Skip()
	}
	fcName := "sequence"
	fn, err := initializePyFunction("inc", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())
	db, err := initializePyFunction("double", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())
	utils.AssertNilMsg(t, err, "failed to initialize function")
	dag, err := fc.CreateSequenceDag(fn, db, fn)
	for _, b := range []bool{true, false} {
		composition := fc.NewFC(fcName, *dag, []*function.Function{fn, db}, b)
		err = composition.SaveToEtcd()
		utils.AssertNil(t, err)

		// verifies the function exists (using function REST API)
		functionNames := getFunctionApiTest(t, HOST, PORT)
		utils.AssertSliceEquals(t, []string{"double", "inc"}, functionNames)

		// verifies the function composition exists (using function composition REST API)
		compositionNames := getCompositionsApiTest(t, HOST, PORT)
		utils.AssertSliceEquals(t, []string{"sequence"}, compositionNames)

		// the API under test is the following
		deleteCompositionApiTest(t, fcName, HOST, PORT)

		// verifies the function composition doen't exists (using function composition REST API)
		compositionNames = getCompositionsApiTest(t, HOST, PORT)
		utils.AssertSliceEquals(t, []string{}, compositionNames)

		functionNames = getFunctionApiTest(t, HOST, PORT)
		if composition.RemoveFnOnDeletion {
			// verifies the function does not exists  (using function REST API)
			utils.AssertSliceEquals(t, []string{}, functionNames)
		} else {
			// verifies the function exists  (using function REST API)
			utils.AssertSliceEquals(t, []string{"double", "inc"}, functionNames)
		}
	}
}

// TestAsyncInvokeComposition tests the REST API that executes a given function composition
func TestAsyncInvokeComposition(t *testing.T) {
	t.Skip() // TODO: Assicurarsi di aspettare la fine della chiamata asincrona
	if !INTEGRATION_TEST {
		t.Skip()
	}
	fcName := "sequence"
	fn, err := initializePyFunction("inc", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())
	utils.AssertNilMsg(t, err, "failed to initialize function")
	dag, err := fc.CreateSequenceDag(fn, fn, fn)
	composition := fc.NewFC(fcName, *dag, []*function.Function{fn}, true)
	createCompositionApiTest(t, &composition, HOST, PORT)

	// verifies the function exists (using function REST API)
	functionNames := getFunctionApiTest(t, HOST, PORT)
	utils.AssertSliceEquals(t, []string{"inc"}, functionNames)

	// === this is the test ===
	params := make(map[string]interface{})
	params["input"] = 1
	invokeCompositionApiTest(t, params, fcName, HOST, PORT, true)
	// TODO wait until the result is available

	// here we do not use REST API
	getFC, b := fc.GetFC(fcName)
	utils.AssertTrue(t, b)
	utils.AssertTrueMsg(t, composition.Equals(getFC), "composition comparison failed")
	err = composition.Delete()
	utils.AssertNilMsg(t, err, "failed to delete composition")

	// verifies the function does not exists  (using function REST API)
	functionNames = getFunctionApiTest(t, HOST, PORT)
	utils.AssertSliceEquals(t, []string{}, functionNames)
}
