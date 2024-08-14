package asl

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
)

type StateMachine struct {
	Name    string
	Comment string
	States  map[string]State
	StartAt string
	Version string
}

// FromASL parses a AWS State Language specification JSON file and returns a StateMachine
// The name of the composition should not be the file name by default, to avoid problems when adding the same composition multiple times.
/*func FromASL(name string, aslSrc []byte) (*fc.FunctionComposition, error) {
	stateMachine, err := parseASL(aslSrc)
	if err != nil {
		return nil, fmt.Errorf("could not parse the ASL file: %v", err)
	}

	startingState, ok := stateMachine.States[stateMachine.StartAt]
	if !ok {
		return nil, fmt.Errorf("could not find starting state")
	}
	// loops until we get to the End
	funcs := make([]*function.Function, 0)
	dag, funcs, err := dagBuilding(funcs, startingState, NewDagBuilder(), nil, stateMachine)
	if err != nil {
		return nil, err
	}

}*/

func (s *StateMachine) getFunctions() []string {
	funcs := make([]string, 0)
	for _, v := range s.States {
		res, ok := v.(HasResources)
		if ok {
			funcName := res.GetResource()
			funcs = append(funcs, funcName)
		}
	}

	return funcs
}

func (s *StateMachine) ToFunctionComposition() (*fc.FunctionComposition, error) {
	dag, err := fc.NewDagBuilder().Build() //TODO: build dag
	if err != nil {
		return nil, fmt.Errorf("could not build DAG: %v", err)
	}

	funcNames := s.getFunctions()
	funcs := make([]*function.Function, 0)
	for _, f := range funcNames {
		funcObj, ok := function.GetFunction(f)
		if !ok {
			return nil, fmt.Errorf("function does not exists")
		}
		funcs = append(funcs, funcObj)
	}

	comp := fc.NewFC(s.Name, *dag, funcs, false)
	return &comp, nil
}
