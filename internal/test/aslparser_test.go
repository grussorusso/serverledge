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

func initializeIncFunction(t *testing.T) *function.Function {
	f, err := InitializePyFunction("inc", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())

	utils.AssertNil(t, err)

	err = f.SaveToEtcd()

	utils.AssertNil(t, err)

	return f
}

// parseFileName takes the name of the file, without .json and parses it. Produces the composition and a single function (for now)
func parseFileName(t *testing.T, aslFileName string) (*fc.FunctionComposition, *function.Function) {
	f := initializeIncFunction(t)

	body, err := os.ReadFile(fmt.Sprintf("asl/%s.json", aslFileName))
	utils.AssertNilMsg(t, err, "unable to read file")

	// for now, we use the same name as the filename to create the composition
	comp, err := fc.FromASL(aslFileName, body)
	utils.AssertNilMsg(t, err, "unable to parse json")
	return comp, f
}

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
	t.Skip("Skipping test")
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	commonTest(t, "simple", 2)
}

// TestParsingSequence verifies that a json with 5 simple nodes is correctly parsed (TODO)
func TestParsingSequence(t *testing.T) {
	t.Skip("Skipping test")
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	commonTest(t, "sequence", 5)

}

// TestParsingMixedUpSequence verifies that a json file with 5 simple unordered task is parsed correctly and in order in a sequence DAG.
func TestParsingMixedUpSequence(t *testing.T) {
	t.Skip("Skipping test")
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

// / TestParsingChoiceFunctionDag verifies that a json file with three different choices is correctly parsed in a Dag with a Choice node and three simple nodes.
func TestParsingChoiceFunctionDag(t *testing.T) {
	t.Skip("Skipping test")
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
	body, err := os.ReadFile("asl/choice.json")
	utils.AssertNilMsg(t, err, "unable to read file")
	// parse the ASL language
	comp, err := fc.FromASL("choice", body)
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
