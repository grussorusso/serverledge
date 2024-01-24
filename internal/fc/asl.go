package fc

import (
	"fmt"
	"github.com/enginyoyen/aslparser"
	"github.com/grussorusso/serverledge/internal/function"
)

func parseASL(aslSrc string) (*aslparser.StateMachine, error) {
	stateMachine, err := aslparser.Parse([]byte(aslSrc), false)
	if err != nil {
		return nil, fmt.Errorf("aslparser failed: %v", err)
	}

	// Seems buggy
	//if !stateMachine.Valid() {
	//	for _, e := range stateMachine.Errors() {
	//		fmt.Print(e.Description())
	//	}
	//	return nil, fmt.Errorf("Invalid ASL file")
	//}

	return stateMachine, nil
}

func FromASL(name string, aslSrc string) (*FunctionComposition, error) {
	stateMachine, err := parseASL(aslSrc)
	if err != nil {
		return nil, fmt.Errorf("could not parse the ASL file: %v", err)
	}

	// TODO: topological sorting
	funcs := make([]*function.Function, 0)
	builder := NewDagBuilder()
	for _, s := range stateMachine.States {
		if s.Type != "Task" {
			return nil, fmt.Errorf("unsupported task type: %s", s.Type) // TODO
		}
		f, found := function.GetFunction(s.Resource)
		if !found {
			return nil, fmt.Errorf("non existing function in composition: %s", s.Resource)
		}
		// get function
		builder = builder.AddSimpleNode(f)
		funcs = append(funcs, f)
		fmt.Printf("Addded simple node with f: %s, funcs=%v", f, funcs)
	}

	dag, err := builder.Build()
	comp := NewFC(name, *dag, funcs, false)
	return &comp, nil
}
