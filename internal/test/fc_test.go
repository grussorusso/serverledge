package test

/// fc_test contains test that executes serverledge server-side function composition apis directly. Internally it uses __function__ REST API.
import (
	"encoding/json"
	"fmt"
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
	u "github.com/grussorusso/serverledge/utils"
	"github.com/lithammer/shortuuid"
	"log"
	"testing"
)

func TestMarshalingFunctionComposition(t *testing.T) {
	fcName := "sequence"
	fn, err := initializePyFunction("inc", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())
	u.AssertNilMsg(t, err, "failed to initialize function")
	dag, err := fc.CreateSequenceDag(fn, fn, fn)
	composition := fc.NewFC(fcName, *dag, []*function.Function{fn}, true)

	marshaledFunc, errMarshal := json.Marshal(composition)
	u.AssertNilMsg(t, errMarshal, "failed to marshal composition")
	var retrieved fc.FunctionComposition
	errUnmarshal := json.Unmarshal(marshaledFunc, &retrieved)
	u.AssertNilMsg(t, errUnmarshal, "failed composition unmarshal")

	u.AssertTrueMsg(t, retrieved.Equals(&composition), fmt.Sprintf("retrieved composition is not equal to initial composition. Retrieved : %s, Expected %s ", retrieved.String(), composition.String()))
}

// TestComposeFC checks the CREATE, GET and DELETE functionality of the Function Composition
func TestComposeFC(t *testing.T) {

	if !INTEGRATION_TEST {
		t.Skip()
	}

	// GET1 - initially we do not have any function composition
	funcs, err := fc.GetAllFC()
	fmt.Println(funcs)
	lenFuncs := len(funcs)
	u.AssertNil(t, err)
	u.AssertEquals(t, 0, lenFuncs)

	fcName := "test"
	// CREATE - we create a test function composition
	m := make(map[string]interface{})
	m["input"] = 0
	length := 3
	_, fArr, err := initializeSameFunctionSlice(length, "js")
	u.AssertNil(t, err)

	dag, err := fc.CreateSequenceDag(fArr...)
	u.AssertNil(t, err)

	fcomp := fc.NewFC(fcName, *dag, fArr, true)
	err2 := fcomp.SaveToEtcd()

	u.AssertNil(t, err2)

	// The creation is successful: we have one more function composition?
	// GET2
	funcs2, err3 := fc.GetAllFC()
	fmt.Println(funcs2)
	u.AssertNil(t, err3)
	u.AssertEqualsMsg(t, lenFuncs+1, len(funcs2), "creation of function failed")

	// the function is exactly the one i created?
	fun, ok := fc.GetFC(fcName)
	u.AssertTrue(t, ok)
	u.AssertTrue(t, fcomp.Equals(fun))

	// DELETE
	err4 := fcomp.Delete()
	u.AssertNil(t, err4)

	// The deletion is successful?
	// GET3
	funcs3, err5 := fc.GetAllFC()
	fmt.Println(funcs3)
	u.AssertNil(t, err5)
	u.AssertEqualsMsg(t, len(funcs3), lenFuncs, "deletion of function failed")
}

// TestInvokeFC executes a Sequential Dag of length N, where each node executes a simple increment function.
func TestInvokeFC(t *testing.T) {

	if !INTEGRATION_TEST {
		t.Skip()
	}

	fcName := "test"
	// CREATE - we create a test function composition
	length := 5
	f, fArr, err := initializeSameFunctionSlice(length, "js")
	u.AssertNil(t, err)
	dag, errDag := fc.CreateSequenceDag(fArr...)
	u.AssertNil(t, errDag)
	fcomp := fc.NewFC(fcName, *dag, fArr, true)
	err1 := fcomp.SaveToEtcd()
	u.AssertNil(t, err1)

	// INVOKE - we call the function composition
	params := make(map[string]interface{})
	params[f.Signature.GetInputs()[0].Name] = 0

	request := fc.NewCompositionRequest(shortuuid.New(), &fcomp, params)

	resultMap, err2 := fcomp.Invoke(request)
	u.AssertNil(t, err2)

	// check result
	output := resultMap.Result[f.Signature.GetOutputs()[0].Name]
	// res, errConv := strconv.Atoi(output.(string))
	u.AssertEquals(t, length, output.(int))
	// u.AssertNil(t, errConv)
	fmt.Printf("%+v\n", resultMap)

	// cleaning up function composition and function
	err3 := fcomp.Delete()
	u.AssertNil(t, err3)

	u.AssertTrueMsg(t, fc.IsEmptyPartialDataCache(), "partial data cache is not empty")
}

// TestInvokeChoiceFC executes a Choice Dag with N alternatives, and it executes only the second one. The functions are all the same increment function
func TestInvokeChoiceFC(t *testing.T) {
	if !INTEGRATION_TEST {
		t.Skip()
	}
	//repeat := 3
	//for i := 0; i < repeat; i++ {
	fcName := "test"
	// CREATE - we create a test function composition
	input := 1
	incJs, errJs := initializeExampleJSFunction()
	u.AssertNil(t, errJs)
	incPy, errPy := initializeExamplePyFunction()
	u.AssertNil(t, errPy)
	doublePy, errDp := initializePyFunction("double", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).Build())
	u.AssertNil(t, errDp)

	dag, errDag := fc.NewDagBuilder().
		AddChoiceNode(
			fc.NewConstCondition(false),
			fc.NewSmallerCondition(2, 1),
			fc.NewConstCondition(true),
		).
		NextBranch(fc.CreateSequenceDag(incJs)).
		NextBranch(fc.CreateSequenceDag(incPy)).
		NextBranch(fc.CreateSequenceDag(doublePy)).
		EndChoiceAndBuild()

	// dag, errDag := fc.CreateChoiceDag(conds, func() (*fc.Dag, error) { return fc.CreateSequenceDag(fArr) })
	u.AssertNil(t, errDag)
	fcomp := fc.NewFC(fcName, *dag, []*function.Function{incJs, incPy, doublePy}, true)
	err1 := fcomp.SaveToEtcd()
	u.AssertNil(t, err1)

	// this is the function that will be called
	f := doublePy

	// INVOKE - we call the function composition
	params := make(map[string]interface{})
	params[f.Signature.GetInputs()[0].Name] = input

	request := fc.NewCompositionRequest(shortuuid.New(), &fcomp, params)
	resultMap, err2 := fcomp.Invoke(request)
	u.AssertNil(t, err2)
	// checking the result, should be input + 1
	output := resultMap.Result[f.Signature.GetOutputs()[0].Name]
	u.AssertEquals(t, input*2, output.(int))
	fmt.Printf("%+v\n", resultMap)

	// cleaning up function composition and function
	err3 := fcomp.Delete()
	u.AssertNil(t, err3)
	//}

	u.AssertTrueMsg(t, fc.IsEmptyPartialDataCache(), "partial data cache is not empty")
}

// TestInvokeFC_DifferentFunctions executes a Sequential Dag of length 2, with two different functions (in different languages)
func TestInvokeFC_DifferentFunctions(t *testing.T) {
	if !INTEGRATION_TEST {
		t.Skip()
	}
	//for i := 0; i < 1; i++ {

	fcName := "test"
	// CREATE - we create a test function composition
	fDouble, errF1 := initializePyFunction("double", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())
	u.AssertNil(t, errF1)

	fInc, errF2 := initializeJsFunction("inc", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())
	u.AssertNil(t, errF2)

	dag, errDag := fc.NewDagBuilder().
		AddSimpleNode(fDouble).
		AddSimpleNode(fInc).
		AddSimpleNode(fDouble).
		AddSimpleNode(fInc).
		Build()

	u.AssertNil(t, errDag)

	fcomp := fc.NewFC(fcName, *dag, []*function.Function{fDouble, fInc}, true)
	err1 := fcomp.SaveToEtcd()
	u.AssertNil(t, err1)

	// INVOKE - we call the function composition
	params := make(map[string]interface{})
	params[fDouble.Signature.GetInputs()[0].Name] = 2
	request := fc.NewCompositionRequest(shortuuid.New(), &fcomp, params)
	resultMap, err2 := fcomp.Invoke(request)
	if err2 != nil {
		log.Printf("%v\n", err2)
		t.FailNow()
	}
	u.AssertNil(t, err2)

	// check result
	output := resultMap.Result[fInc.Signature.GetOutputs()[0].Name]
	if output != 11 {
		t.FailNow()
	}

	// res, errConv := strconv.Atoi(output.(string))
	u.AssertEquals(t, (2*2+1)*2+1, output.(int))
	// u.AssertNil(t, errConv)
	fmt.Println(resultMap)

	// cleaning up function composition and function
	err3 := fcomp.Delete()
	u.AssertNil(t, err3)
	//}
	u.AssertTrueMsg(t, fc.IsEmptyPartialDataCache(), "partial data cache is not empty")
}

// TestInvokeFC_BroadcastFanOut executes a Parallel Dag with N parallel branches
func TestInvokeFC_BroadcastFanOut(t *testing.T) {
	t.Skip() // TODO: correggere
	if !INTEGRATION_TEST {
		t.Skip()
	}
	//for i := 0; i < 1; i++ {

	fcName := "test"
	// CREATE - we create a test function composition
	fDouble, errF1 := initializePyFunction("double", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())
	u.AssertNil(t, errF1)

	fInc, errF2 := initializeJsFunction("inc", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())
	u.AssertNil(t, errF2)

	width := 3
	dag, errDag := fc.CreateBroadcastDag(func() (*fc.Dag, error) { return fc.CreateSequenceDag(fDouble) }, width)
	u.AssertNil(t, errDag)
	dag.Print()

	fcomp := fc.NewFC(fcName, *dag, []*function.Function{fDouble, fInc}, true)
	err1 := fcomp.SaveToEtcd()
	u.AssertNil(t, err1)

	// INVOKE - we call the function composition
	params := make(map[string]interface{})
	params[fDouble.Signature.GetInputs()[0].Name] = 1
	request := fc.NewCompositionRequest(shortuuid.New(), &fcomp, params)
	resultMap, err2 := fcomp.Invoke(request)
	u.AssertNil(t, err2)

	// check multiple result
	output := resultMap.Result[fInc.Signature.GetOutputs()[0].Name]
	u.AssertNonNil(t, output) // FIXME: fanin output is null!
	for _, res := range output.(map[string]interface{}) {
		u.AssertEquals(t, 2, res.(int))
	}
	// u.AssertNil(t, errConv)
	fmt.Println(resultMap)

	// cleaning up function composition and functions
	err3 := fcomp.Delete()
	u.AssertNil(t, err3)

	u.AssertTrueMsg(t, fc.IsEmptyPartialDataCache(), "partial data cache is not empty")
}
