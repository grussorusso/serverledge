package test

import (
	"testing"

	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/utils"
)

func TestExperiment4(t *testing.T) {
	/*if !Experiment {
		t.Skip()
	}*/
	_, err := createParallelComposition(t)
	utils.AssertNilMsg(t, err, "failed to create composition")

	// err2 := comp.Delete()
	// utils.AssertNilMsg(t, err2, "failed to delete composition and functions")
}

/*
 * This test will execute the following Dag:
 *		        Start
 *		          |
 *		        Choice
 * Task==1      Task==2     true
 *    |            |          |
 *   inc         double     Fan Out
 *    |            |        |     |
 *    |            |       inc   inc
 *    |            |        |     |
 *    |            |        Fan In
 *    |            |           |
 *    ------------End-----------
 */
func createParallelComposition(t *testing.T) (*fc.FunctionComposition, error) {

	fnInc, err := InitializePyFunction("inc", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddInput("Task", function.Int{}). // should be true
		AddOutput("result", function.Int{}).
		Build())

	fnDouble, err := InitializePyFunction("double", "handler", function.NewSignature().
		AddInput("input", function.Int{}).
		AddInput("Task", function.Int{}). // should be false
		AddOutput("result", function.Int{}).
		Build())

	dag, err := fc.NewDagBuilder().
		AddChoiceNode(
			fc.NewEqParamCondition(fc.NewParam("Task"), fc.NewValue(1)),
			fc.NewEqParamCondition(fc.NewParam("Task"), fc.NewValue(2)),
			fc.NewConstCondition(true),
		).
		NextBranch(fc.CreateSequenceDag(fnInc)).
		NextBranch(fc.CreateSequenceDag(fnDouble)).
		NextBranch(fc.NewDagBuilder().
			AddScatterFanOutNode(2).
			ForEachParallelBranch(fc.LambdaSequenceDag(fnInc)).
			AddFanInNode(fc.AddToArrayEntry).
			Build()).
		EndChoiceAndBuild()

	composition, err := fc.NewFC("complex", *dag, []*function.Function{fnInc, fnDouble}, true)
	utils.AssertNil(t, err)
	createCompositionApiTest(t, composition, "127.0.0.1", 1323)
	return composition, nil
}
