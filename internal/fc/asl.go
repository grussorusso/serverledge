package fc

/*** Adapted from https://github.com/enginyoyen/aslparser ***/
import (
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/grussorusso/serverledge/internal/asl"
	"github.com/grussorusso/serverledge/internal/function"
	"strconv"
)

// DEPRECATED
type Retry struct {
	ErrorEquals     []string
	IntervalSeconds int
	BackoffRate     int
	MaxAttempts     int
}

// DEPRECATED
type Catch struct {
	ErrorEquals []string
	ResultPath  string
	Next        string
}

// State implements a state for Amazon state language TODO: separate states in single files and make State an interface
// DEPRECATED
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
	Choices          []Match
}

// DEPRECATED
type Match struct {
	Variable  string
	Operation Condition
	Next      string
}

// DEPRECATED
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
		switch string(stateType) {
		case "Task":
			stateResource, _, _, errTask := jsonparser.Get(value, "Resource")
			if errTask != nil {
				return errTask
			}
			stateEnd, dType, _, errTask := jsonparser.Get(value, "End")
			if dType != jsonparser.NotExist && errTask != nil {
				return errTask
			} // Retrieve the next state, otherwise return nil
			if dType == jsonparser.NotExist || string(stateEnd) != "true" {
				nextState, _, _, errTask := jsonparser.Get(value, "Next")
				if errTask != nil {
					return errTask
				}
				s := State{Name: string(key), Type: string(stateType), Resource: string(stateResource), Next: string(nextState)}
				states[s.Name] = s
				fmt.Println("Created state: ", s.Name)
			}
			return nil
		case "Choice":
			choices, _, _, errChoice := jsonparser.Get(value, "Choices")
			if errChoice != nil {
				return errChoice
			}
			matches := make([]Match, 0)

			_, errArr := jsonparser.ArrayEach(choices, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
				matchVariable, _, _, errMatch := jsonparser.Get(value, "Variable")
				if errMatch != nil {
					return
				}
				matchCondition, _, _, errMatch := jsonparser.Get(value, "NumericEquals") // TODO: only checks NumericEquals
				if errMatch != nil {
					return
				}
				num, errNum := strconv.Atoi(string(matchCondition))
				if errNum != nil {
					return
				}
				matchNext, _, _, errMatch := jsonparser.Get(value, "Next")
				if errMatch != nil {
					return
				}
				m := Match{
					Variable:  string(matchVariable),
					Operation: NewEqCondition(NewParam(string(matchVariable)), NewValue(num)),
					Next:      string(matchNext),
				}
				matches = append(matches, m)
			})
			if errArr != nil {
				return errArr
			}

			defaultMatch, _, _, errChoice := jsonparser.Get(value, "Default")
			if errChoice != nil {
				return err
			}

			s := State{Name: string(key), Type: string(stateType), Choices: matches, Default: string(defaultMatch)}
			states[s.Name] = s
			fmt.Println("Created state: ", s.Name)
			//"Choices": [
			//	{
			//		"Variable": "$.input",
			//		"NumericEquals": 1,
			//		"Next": "FirstMatchState"
			//	},
			//	{
			//		"Variable": "$.input",
			//		"NumericEquals": 2,
			//		"Next": "SecondMatchState"
			//	}
			//],
			//"Default": "DefaultState"
			break
		case "Fail":
			fmt.Println("Created state fail")
			return nil
		default:
			return fmt.Errorf("invalid ASL: unknown state '%s'", string(stateType))
		}
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
	stateMachine, err := asl.ParseFrom(name, aslSrc)
	if err != nil {
		return nil, fmt.Errorf("could not parse the ASL file: %v", err)
	}
	return stateMachine.ToFunctionComposition(false)
	//startingState, ok := stateMachine.States[stateMachine.StartAt]
	//if !ok {
	//	return nil, fmt.Errorf("could not find starting state")
	//}
	//// loops until we get to the End
	//funcs := make([]*function.Function, 0)
	//dag, funcs, err := dagBuilding(funcs, startingState, NewDagBuilder(), nil, stateMachine)
	//if err != nil {
	//	return nil, err
	//}
	//
	//comp := NewFC(name, *dag, funcs, false)
	//return &comp, nil
}

func dagBuilding(funcs []*function.Function, currentState State, builder *DagBuilder, condBuilder *ChoiceBranchBuilder, stateMachine *StateMachine) (*Dag, []*function.Function, error) {
	switch currentState.Type {
	case "Task":
		f, found := function.GetFunction(currentState.Resource)
		if !found {
			return nil, nil, fmt.Errorf("non existing function in composition: %s", currentState.Resource)
		}
		builder = builder.AddSimpleNodeWithId(f, currentState.Name)
		funcs = append(funcs, f)
		fmt.Printf("Added simple node with f: %s, funcs=%v\n", f, funcs)

		// if we are at the end, we close it
		nextState, ok := stateMachine.States[currentState.Next]
		if !ok && currentState.End == false {
			// we have an error
			return nil, funcs, fmt.Errorf("could not find next state in composition: %s", currentState.Next)
		} else if ok && currentState.End == true {
			// we ended the parsing correctly
			dag, err := builder.Build()
			return dag, funcs, err
		} else {
			// we should continue to parse
			return dagBuilding(funcs, nextState, builder, nil, stateMachine)
		}

	case "Choice":
		// gets the conditions from matches
		conds := make([]Condition, 0)
		for _, c := range currentState.Choices {
			conds = append(conds, c.Operation)
		}
		if condBuilder == nil {
			condBuilder = builder.AddChoiceNode(conds...)
		}

		i := 0
		for condBuilder.HasNextBranch() {
			nextState := stateMachine.States[currentState.Choices[i].Next]
			subDag, _, errSub := dagBuilding(funcs, nextState, NewDagBuilder(), condBuilder, stateMachine)
			if errSub != nil {
				return nil, nil, errSub
			}
			// assign the function (Task state) to the next branch
			condBuilder.NextBranch(subDag, nil)
			i++
		}
		dag, err := condBuilder.EndChoiceAndBuild()
		return dag, funcs, err
	case "Fail":
		dag, err := builder.Build()
		return dag, funcs, err
	default:
		return nil, nil, fmt.Errorf("unsupported task type: %s", currentState.Type) // TODO
	}
}
