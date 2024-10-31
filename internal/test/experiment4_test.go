package test

import (
	"fmt"
	"testing"

	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/utils"
)

// create a sequence of varying length and run the experiment for 10 minutes

func TestExperiment4(t *testing.T) {
	/*
		if !Experiment {
			t.Skip()
		}*/
	lengths := []int{1, 2, 4, 8, 16, 32}

	for _, length := range lengths {
		_, err := CreateIncCompositionSequence(t, fmt.Sprintf("sequence_inc_%d", length), "localhost", 1323, length)
		utils.AssertNilMsg(t, err, "failed to create composition")

	}

	// err2 := comp.Delete()
	// utils.AssertNilMsg(t, err2, "failed to delete composition and functions")

}

// TestCreateComposition tests the compose REST API that creates a new function composition
func CreateIncCompositionSequence(t *testing.T, fcName string, host string, port int, length int) (*fc.FunctionComposition, error) {
	fn, err := InitializePyFunction("inc", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddOutput("output", function.Int{}).
		Build())
	utils.AssertNilMsg(t, err, "failed to initialize function noop")

	fArr := make([]*function.Function, 0, length)
	for i := 0; i < length; i++ {
		fArr = append(fArr, fn)
	}

	dag, err := fc.CreateSequenceDag(fArr...)
	utils.AssertNil(t, err)
	composition, err := fc.NewFC(fcName, *dag, []*function.Function{fn}, false)
	utils.AssertNil(t, err)
	createCompositionApiTest(t, composition, host, port)

	return composition, nil
}
