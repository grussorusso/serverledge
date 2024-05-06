package test

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/utils"
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

// / name of composition should be the same as the filename (without extension)
func TestParsedCompositionName(t *testing.T) {
	// This does not check the value, the only important thing is to define the INTEGRATION environment variable
	if !IntegrationTest {
		t.Skip()
	}
	expectedName := "simple"
	comp, _ := parseFileName(t, expectedName)
	// the name should be simple, because we parsed the "simple.json" file
	utils.AssertEquals(t, comp.Name, expectedName)
}

// TestParsingSimple verifies that a simple json with 2 state is correctly parsed and it is equal to a sequence dag with 2 simple nodes
func TestParsingSimple(t *testing.T) {
	// This does not check the value, the only important thing is to define the INTEGRATION environment variable
	if !IntegrationTest {
		t.Skip()
	}
	// check the number of workflows before creating the new one
	all, err := fc.GetAllFC()
	utils.AssertNil(t, err)

	comp, _ := parseFileName(t, "simple")

	err = comp.SaveToEtcd()
	utils.AssertNilMsg(t, err, "unable to save parsed composition")

	all2, err := fc.GetAllFC()
	utils.AssertNil(t, err)
	utils.AssertTrue(t, len(all2) == len(all)+1)

	expectedComp, ok := fc.GetFC("simple")
	utils.AssertTrue(t, ok)

	utils.AssertTrueMsg(t, comp.Equals(expectedComp), "parsed composition differs from expected composition")
}

// TestParsingSequence verifies that a json with 5 simple nodes is correctly parsed (TODO)
func TestParsingSequence(t *testing.T) {
	// This does not check the value, the only important thing is to define the INTEGRATION environment variable
	if !IntegrationTest {
		t.Skip()
	}
	body, err := os.ReadFile("asl/sequence.json")
	utils.AssertNilMsg(t, err, "unable to read file")

	sm, _ := fc.FromASL("simple", body)
	fmt.Printf("Found state machine:  %v\n", sm)

	fmt.Println()
}
