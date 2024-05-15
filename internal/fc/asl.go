package fc

/*** Adapted from https://github.com/enginyoyen/aslparser ***/
import (
	"fmt"

	"github.com/buger/jsonparser"
	"github.com/grussorusso/serverledge/internal/function"
)

type Retry struct {
	ErrorEquals     []string
	IntervalSeconds int
	BackoffRate     int
	MaxAttempts     int
}

type Catch struct {
	ErrorEquals []string
	ResultPath  string
	Next        string
}

// State implements a state for Amazon state language
type State struct {
	Name             string
	Comment          string
	Type             string
	Next             string
	Default          string
	Resource         string
	End              bool
	Parameters       map[string]interface{}
	Retry            []Retry
	Catch            []Catch
	TimeoutSeconds   int
	HeartbeatSeconds int
}

type StateMachine struct {
	Comment string
	States  map[string]State
	StartAt string
	Version string
	// validationResult *gojsonschema.Result
}

func parseASL(aslSrc []byte) (*StateMachine, error) {
	//stateMachine, err := aslparser.Parse([]byte(aslSrc), false)
	//if err != nil {
	//	return nil, fmt.Errorf("aslparser failed: %v", err)
	//}
	initialState, _, _, err := jsonparser.Get(aslSrc, "StartAt")
	if err != nil {
		return nil, fmt.Errorf("invalid ASL: missing StartAt key")
	}

	statesData, _, _, err := jsonparser.Get(aslSrc, "States")
	if err != nil {
		return nil, fmt.Errorf("invalid ASL: missing States key")
	}

	states := make(map[string]State, 0)
	err = jsonparser.ObjectEach(statesData, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		stateType, _, _, err2 := jsonparser.Get(value, "Type")
		if err2 != nil {
			return err2
		}
		stateResource, _, _, err2 := jsonparser.Get(value, "Resource")
		if err2 != nil {
			return err2
		}
		nextState, _, _, err2 := jsonparser.Get(value, "Next")
		s := State{Name: string(key), Type: string(stateType), Resource: string(stateResource), Next: string(nextState)}
		states[s.Name] = s
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Invalid ASL: %v", err)
	}

	stateMachine := &StateMachine{States: states, StartAt: string(initialState)}
	fmt.Printf("Found state machine:  %v\n", stateMachine)

	return stateMachine, nil
}

// FromASL parses a AWS State Language specification file and returns a Function Composition with the corresponding Serverledge Dag
// The name of the composition should not be the file name by default, to avoid problems when adding the same composition multiple times.
func FromASL(name string, aslSrc []byte) (*FunctionComposition, error) {
	stateMachine, err := parseASL(aslSrc)
	if err != nil {
		return nil, fmt.Errorf("could not parse the ASL file: %v", err)
	}

	//adj := make(map[string][]string)
	//for k, v := range stateMachine.States {
	//	if v.Type == "Task" {
	//		adj[k] = make([]string, 1)
	//		adj[k] = append(adj[k], v.Next)
	//	} else if v.Type == "Choice" {
	//		adj[k] = make([]string, 1)
	//		fmt.Printf("Raw: %v\n", v)
	//		fmt.Printf("Next: %v\n", v.Next)
	//		// TODO: adj[k] = append(adj[k], v.Default)
	//	}
	//}
	//fmt.Println(adj)

	// TODO: topological sorting

	currentState, ok := stateMachine.States[stateMachine.StartAt]

	// loops until we get to the End
	funcs := make([]*function.Function, 0)
	builder := NewDagBuilder()
	for ok {
		// TODO support other types of States
		if currentState.Type != "Task" {
			return nil, fmt.Errorf("unsupported task type: %s", currentState.Type) // TODO
		}

		f, found := function.GetFunction(currentState.Resource)
		if !found {
			return nil, fmt.Errorf("non existing function in composition: %s", currentState.Resource)
		}
		builder = builder.AddSimpleNodeWithId(f, currentState.Name)
		funcs = append(funcs, f)
		fmt.Printf("Added simple node with f: %s, funcs=%v\n", f, funcs)

		// if we are at the end, we close it
		currentState, ok = stateMachine.States[currentState.Next]
	}

	dag, err := builder.Build()
	if err != nil {
		return nil, err
	}

	comp := NewFC(name, *dag, funcs, false)
	return &comp, nil
}
