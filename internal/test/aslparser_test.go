package test

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/utils"
	"github.com/lithammer/shortuuid"
	"os"
	"testing"
)

// / TestParsedCompositionName verifies that the composition name matches the filename (without extension)
func TestParsedCompositionName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	expectedName := "simple"
	comp, _ := parseFileName(t, expectedName)
	// the name should be simple, because we parsed the "simple.json" file
	utils.AssertEquals(t, comp.Name, expectedName)
}

// commonTest creates a function, parses a json AWS State Language file producing a function composition,
// then checks if the composition is saved onto ETCD. Lastly, it runs the composition and expects the correct result.
func commonTest(t *testing.T, name string, expectedResult int) {
	all, err := fc.GetAllFC()
	utils.AssertNil(t, err)

	comp, f := parseFileName(t, name)
	defer func() {
		err = comp.Delete()
		utils.AssertNilMsg(t, err, "failed to delete composition")
	}()
	// saving to etcd is not necessary to run the function composition, but is needed when offloading
	{
		err := comp.SaveToEtcd()
		utils.AssertNilMsg(t, err, "unable to save parsed composition")

		all2, err := fc.GetAllFC()
		utils.AssertNil(t, err)
		utils.AssertEqualsMsg(t, len(all2), len(all)+1, "the number of created functions differs")

		expectedComp, ok := fc.GetFC(name)
		utils.AssertTrue(t, ok)

		utils.AssertTrueMsg(t, comp.Equals(expectedComp), "parsed composition differs from expected composition")
		fmt.Println(comp)
	}

	// runs the workflow
	params := make(map[string]interface{})
	params[f.Signature.GetInputs()[0].Name] = 0
	request := fc.NewCompositionRequest(shortuuid.New(), comp, params)
	resultMap, err2 := comp.Invoke(request)
	utils.AssertNil(t, err2)

	// checks the result
	output := resultMap.Result[f.Signature.GetOutputs()[0].Name]
	utils.AssertEquals(t, expectedResult, output.(int))
	fmt.Println("Result: ", output)
}

// TestParsingSimple verifies that a simple json with 2 state is correctly parsed and it is equal to a sequence dag with 2 simple nodes

func TestParsingSimple(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	commonTest(t, "simple", 2)
}

// TestParsingSequence verifies that a json with 5 simple nodes is correctly parsed
func TestParsingSequence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	commonTest(t, "sequence", 5)

}

// TestParsingMixedUpSequence verifies that a json file with 5 simple unordered task is parsed correctly and in order in a sequence DAG.
func TestParsingMixedUpSequence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	commonTest(t, "mixed_sequence", 5)
}

// / TestParsingMultipleFunctionSequence verifies that a json file with three different functions is correctly parsed from a DAG.
func TestParsingMultipleFunctionSequence(t *testing.T) {

	// TODO write json file with multiple functions
	//all, err := fc.GetAllFC()
	//utils.AssertNil(t, err)
	//
	//comp, f := parseFileName(t, name)
	//defer func() {
	//	err = comp.Delete()
	//	utils.AssertNilMsg(t, err, "failed to delete composition")
	//}()
	//// saving to etcd is not necessary to run the function composition, but is needed when offloading
	//{
	//	err := comp.SaveToEtcd()
	//	utils.AssertNilMsg(t, err, "unable to save parsed composition")
	//
	//	all2, err := fc.GetAllFC()
	//	utils.AssertNil(t, err)
	//	utils.AssertEqualsMsg(t, len(all2), len(all)+1, "the number of created functions differs")
	//
	//	expectedComp, ok := fc.GetFC(name)
	//	utils.AssertTrue(t, ok)
	//
	//	utils.AssertTrueMsg(t, comp.Equals(expectedComp), "parsed composition differs from expected composition")
	//	fmt.Println(comp)
	//}
	//
	//// runs the workflow
	//params := make(map[string]interface{})
	//params[f.Signature.GetInputs()[0].Name] = 0
	//request := fc.NewCompositionRequest(shortuuid.New(), comp, params)
	//resultMap, err2 := comp.Invoke(request)
	//utils.AssertNil(t, err2)
	//
	//// checks the result
	//output := resultMap.Result[f.Signature.GetOutputs()[0].Name]
	//utils.AssertEquals(t, expectedResult, output.(int))
	//fmt.Println("Result: ", output)
	// TODO run the dag end expect the correct result
}

// / TestParsingChoiceFunctionDagWithDefaultFail verifies that a json file with three different choices is correctly parsed in a Dag with a Choice node and three simple nodes.
func TestParsingChoiceFunctionDagWithDefaultFail(t *testing.T) {
	t.Skip("fail is not implemented")
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Creates "inc", "double" and "hello" python functions
	fns := initializeAllPyFunctionFromNames(t, "inc", "double", "hello")

	incFn := fns[0]
	//doubleFn := fns[1]
	helloFn := fns[2]

	// reads the file
	body, err := os.ReadFile("asl/choice_numeq_fail.json")
	utils.AssertNilMsg(t, err, "unable to read file")
	// parse the ASL language
	comp, err := fc.FromASL("choice", body) // TODO: implement fail parsing
	utils.AssertNilMsg(t, err, "unable to parse json")

	// runs the workflow
	params := make(map[string]interface{})
	params[incFn.Signature.GetInputs()[0].Name] = 0
	request := fc.NewCompositionRequest(shortuuid.New(), comp, params)
	resultMap, err2 := comp.Invoke(request)
	utils.AssertNil(t, err2)

	// checks the result
	output := resultMap.Result[helloFn.Signature.GetOutputs()[0].Name]
	utils.AssertEquals(t, "expectedResult", output.(string))
	fmt.Println("Result: ", output)
}

// 1st branch (input==1): inc + inc (expected nothing)
// 2nd branch (input==2): double + inc (expected nothing)
// def branch (true    ): hello (expected nothing)
func TestParsingChoiceDagWithDataTestExpr(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	// Creates "inc", "double" and "hello" python functions
	funcs := initializeAllPyFunctionFromNames(t, "inc", "double", "hello")

	// reads the file
	body, err := os.ReadFile("asl/choice_datatestexpr.json")
	utils.AssertNilMsg(t, err, "unable to read file")
	// parse the ASL language
	comp, err := fc.FromASL("choice3", body)
	utils.AssertNilMsg(t, err, "unable to parse json")

	incFn := funcs[0]
	// helloFn := funcs[len(funcs)-1]

	// runs the workflow (1st choice branch) // TODO: first branch should simply use inc
	//params := make(map[string]interface{})
	//params[incFn.Signature.GetInputs()[0].Name] = 0
	//request := fc.NewCompositionRequest(shortuuid.New(), comp, params)
	//_, err2 := comp.Invoke(request) // TODO: Default state fails the nextState is the same SimpleNode, but has the same name in the state machine
	//utils.AssertNil(t, err2)

	// checks the result // TODO: check that output is 1+1+1=3
	//output := resultMap.Result[helloFn.Signature.GetOutputs()[0].Name]
	//utils.AssertEquals(t, "expectedResult", output.(string))
	//fmt.Println("Result: ", output)

	// runs the workflow (2nd choice branch) // TODO: second branch should use double and then inc
	//params := make(map[string]interface{})
	//params[incFn.Signature.GetInputs()[0].Name] = 0
	//request := fc.NewCompositionRequest(shortuuid.New(), comp, params)
	//resultMap, err2 := comp.Invoke(request) // TODO: Default state fails the nextState is the same SimpleNode, but has the same name in the state machine
	//utils.AssertNil(t, err2)

	// checks the result // TODO: check that output is 2*2+1 = 5
	//output := resultMap.Result[helloFn.Signature.GetOutputs()[0].Name]
	//utils.AssertEquals(t, "expectedResult", output.(string))
	//fmt.Println("Result: ", output)

	// runs the workflow (default choice branch) // TODO: should only print hello
	paramsDefault := make(map[string]interface{})
	paramsDefault[incFn.Signature.GetInputs()[0].Name] = "Giacomo"
	request := fc.NewCompositionRequest(shortuuid.New(), comp, paramsDefault)
	resultMap, errDef := comp.Invoke(request) // TODO: Default state fails the nextState is the same SimpleNode, but has the same name in the state machine
	utils.AssertNil(t, errDef)
	fmt.Printf("Composition Execution Report: %s\n", resultMap.String())

	// checks the result // TODO: should check that contains the printed result.
	//output := resultMap.Result[helloFn.Signature.GetOutputs()[0].Name]
	//utils.AssertEquals(t, "expectedResult", output.(string))
	//fmt.Println("Result: ", output)
}

func TestParsingChoiceDagWithBoolExpr(t *testing.T) {
	t.Skip("WIP")
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Creates "inc", "double" and "hello" python functions
	incFn, err := InitializePyFunction("inc", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())
	doubleFn, err := InitializePyFunction("double", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())
	helloFn, err := InitializePyFunction("hello", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())
	// Removes the functions after test execution
	defer func() {
		err := incFn.Delete()
		utils.AssertNil(t, err)
		err = doubleFn.Delete()
		utils.AssertNil(t, err)
		err = helloFn.Delete()
		utils.AssertNil(t, err)
	}()

	err = incFn.SaveToEtcd()
	utils.AssertNilMsg(t, err, "failed to create inc fn")
	err = doubleFn.SaveToEtcd()
	utils.AssertNilMsg(t, err, "failed to create double fn")
	err = helloFn.SaveToEtcd()
	utils.AssertNilMsg(t, err, "failed to create hello fn")

	// reads the file
	body, err := os.ReadFile("asl/choice_boolexpr.json")
	utils.AssertNilMsg(t, err, "unable to read file")
	// parse the ASL language
	comp, err := fc.FromASL("choice2", body)
	utils.AssertNilMsg(t, err, "unable to parse json")
	// deletes the composition after test execution
	defer func() {
		err = comp.Delete()
		utils.AssertNilMsg(t, err, "failed to delete composition")
	}()

	// runs the workflow
	params := make(map[string]interface{})
	params[incFn.Signature.GetInputs()[0].Name] = 0
	request := fc.NewCompositionRequest(shortuuid.New(), comp, params)
	resultMap, err2 := comp.Invoke(request)
	utils.AssertNil(t, err2)

	// checks the result
	output := resultMap.Result[helloFn.Signature.GetOutputs()[0].Name]
	utils.AssertEquals(t, "expectedResult", output.(string))
	fmt.Println("Result: ", output)
}

/*
&{
	map[
		ChoiceState:{ChoiceState  Choice  DefaultState  false map[] [] [] 0 0 [
			{input {4 [false false] [0xc00017e6f0 0xc00017e720] []} FirstMatchState}
			{input {4 [false false] [0xc00017e750 0xc00017e780] []} SecondMatchState}
		]}
		] FirstState }
*/

func TestParsingDagWithMalformedJson(t *testing.T) {}

func TestParsingDagWithUnknownFunction(t *testing.T) {}
