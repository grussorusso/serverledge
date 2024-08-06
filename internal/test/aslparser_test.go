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
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	commonTest(t, "simple", 2)
}

// TestParsingSequence verifies that a json with 5 simple nodes is correctly parsed (TODO)
func TestParsingSequence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	commonTest(t, "sequence", 5)

}

// TestParsingMixedUpSequence verifies that a json file with 5 simple unordered task is parsed correctly and in order in a sequence DAG. TODO: create a
func TestParsingMixedUpSequence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	commonTest(t, "mixed_sequence", 5)
}

// / TestParsingMultipleFunctionSequence verifies that a json file with three different functions is correctly parsed from a DAG.
func TestParsingMultipleFunctionSequence(t *testing.T) {
	// TODO write json file with multiple functions
	// TODO run the dag end expect the correct result
}

// / TestParsingChoiceFunctionDag verifies that a json file with three different choices is correctly parsed in a Dag with a Choice node and three simple nodes.
func TestParsingChoiceFunctionDag(t *testing.T) {
	// TODO write json file with Choice Task
	// TODO run the dag end expect the correct result
}
