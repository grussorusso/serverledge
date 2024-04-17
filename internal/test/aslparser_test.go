package test

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/utils"
	"os"
	"testing"
)

func initializeIncFunction(t *testing.T) {
	f, err := InitializePyFunction("inc", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("result", function.Int{}).
		Build())

	utils.AssertNil(t, err)

	err = f.SaveToEtcd()

	utils.AssertNil(t, err)
}

func parseFile(t *testing.T) *fc.FunctionComposition {
	initializeIncFunction(t)

	body, err := os.ReadFile("../../test/simple.json")
	utils.AssertNilMsg(t, err, "unable to read file")

	comp, err := fc.FromASL("prova", body)
	utils.AssertNilMsg(t, err, "unable to parse json")
	return comp
}

func TestParsingSimple(t *testing.T) {
	// This does not check the value, the only important thing is to define the INTEGRATION environment variable
	if !IntegrationTest {
		t.Skip()
	}

	comp := parseFile(t)

	fmt.Println(comp)
	err := comp.SaveToEtcd()
	utils.AssertNilMsg(t, err, "unable to save parsed composition")

	all, err := fc.GetAllFC()
	utils.AssertNil(t, err)

	fmt.Println(all)
}

func TestParsingSequence(t *testing.T) {
	// This does not check the value, the only important thing is to define the INTEGRATION environment variable
	if !IntegrationTest {
		t.Skip()
	}
	body, err := os.ReadFile("../../test/sequence.json")
	utils.AssertNilMsg(t, err, "unable to read file")

	sm, _ := fc.FromASL("prova", body)
	fmt.Printf("Found state machine:  %v\n", sm)

	fmt.Println()
}
