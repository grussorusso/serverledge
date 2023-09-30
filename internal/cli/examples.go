package cli

import (
	"encoding/base64"
	"fmt"
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
)

func exampleParsing(str string) (*fc.Dag, []*function.Function, error) {

	py, err := InitializePyFunction("inc", "handler", function.NewSignature().AddInput("input", function.Int{}).AddOutput("result", function.Int{}).Build())
	if err != nil {
		return nil, nil, err
	}

	switch str {
	case "sequence":
		dag, errSequence := fc.CreateSequenceDag(py, py, py)
		return dag, []*function.Function{py}, errSequence
	case "choice":
		dag, errChoice := fc.CreateChoiceDag(fc.LambdaSequenceDag(py, py), fc.NewConstCondition(false), fc.NewConstCondition(true))
		return dag, []*function.Function{py}, errChoice
	case "parallel":
		dag, errParallel := fc.CreateBroadcastDag(fc.LambdaSequenceDag(py, py), 3)
		return dag, []*function.Function{py}, errParallel
	case "complex":
		dag, errComplex := fc.NewDagBuilder().
			AddSimpleNode(py).
			AddChoiceNode(fc.NewEqCondition(1, 4), fc.NewDiffCondition(1, 4)).
			NextBranch(fc.CreateSequenceDag(py)).
			NextBranch(fc.NewDagBuilder().
				AddScatterFanOutNode(3).
				ForEachParallelBranch(fc.LambdaSequenceDag(py)).
				AddFanInNode(fc.AddToArrayEntry).
				Build()).
			EndChoiceAndBuild()
		return dag, []*function.Function{py}, errComplex
	default:
		return nil, nil, fmt.Errorf("failed to parse dag - use a default dag like 'sequence', 'choice', 'parallel' or 'complex'")
	}
}

func InitializePyFunction(name string, handler string, sign *function.Signature) (*function.Function, error) {
	srcPath := "./examples/" + name + ".py"
	srcContent, err := ReadSourcesAsTar(srcPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read python sources %s as tar: %v", srcPath, err)
	}
	encoded := base64.StdEncoding.EncodeToString(srcContent)
	PY_MEMORY := int64(20)
	f := function.Function{
		Name:            name,
		Runtime:         "python310",
		MemoryMB:        PY_MEMORY,
		CPUDemand:       1.0,
		Handler:         fmt.Sprintf("%s.%s", name, handler), // on python, for now is needed file name and handler name!!
		TarFunctionCode: encoded,
		Signature:       sign,
	}
	return &f, nil
}

func InitializeJsFunction(name string, sign *function.Signature) (*function.Function, error) {
	srcPath := "./examples/" + name + ".js"
	srcContent, err := ReadSourcesAsTar(srcPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read js sources %s as tar: %v", srcPath, err)
	}
	encoded := base64.StdEncoding.EncodeToString(srcContent)
	JS_MEMORY := int64(50)
	f := function.Function{
		Name:            name,
		Runtime:         "nodejs17ng",
		MemoryMB:        JS_MEMORY,
		CPUDemand:       1.0,
		Handler:         name, // on js only file name is needed!!
		TarFunctionCode: encoded,
		Signature:       sign,
	}
	return &f, nil
}
