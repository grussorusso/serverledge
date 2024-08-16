package asl

import (
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/grussorusso/serverledge/internal/types"
	"github.com/grussorusso/serverledge/utils"
)

type StateMachine struct {
	Name    string
	Comment string // Optional
	States  map[string]State
	StartAt string
	Version string // Optional
}

func ParseFrom(name string, aslSrc []byte) (*StateMachine, error) {

	startAt, err := utils.JsonExtract(aslSrc, "StartAt")
	if err != nil {
		return nil, fmt.Errorf("invalid ASL: missing StartAt key")
	}

	comment := utils.JsonExtractStringOrDefault(aslSrc, "Comment", "")

	version := utils.JsonExtractStringOrDefault(aslSrc, "Version", "1.0")

	statesData, err := utils.JsonExtract(aslSrc, "States")
	if err != nil {
		return nil, fmt.Errorf("invalid ASL: missing States key")
	}

	statesMap, err := parseStates(statesData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse States key: %v", err)
	}

	sm := StateMachine{
		Name:    name,
		StartAt: startAt,
		Comment: comment,
		Version: version,
		States:  statesMap,
	}

	return &sm, nil
}

func emptyParsableFromType(t StateType) Parseable {
	switch t {
	case Task:
		return NewEmptyTask()
	case Choice:
		return NewEmptyChoice()
	case Parallel:
		return NewEmptyParallel()
	case Map:
		return NewEmptyMap()
	case Pass:
		return NewEmptyPass()
	case Wait:
		return NewEmptyWait()
	case Succeed:
		return NewEmptySucceed()
	case Fail:
		return NewEmptyFail()
	default:
		return nil
	}
}

// parseStates is used to create the state machine
func parseStates(statesData string) (map[string]State, error) {
	states := make(map[string]State)
	err := jsonparser.ObjectEach([]byte(statesData), func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		stateType, err2 := utils.JsonExtract(value, "Type")
		if err2 != nil {
			return err2
		}

		parseable := emptyParsableFromType(StateType(stateType))
		parsedState, err := parseable.ParseFrom(value)
		if err != nil {
			return err
		}
		states[string(key)] = parsedState
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid ASL: %v", err)
	}

	return states, nil
}

func (sm *StateMachine) String() string {

	statesString := "["
	for key, state := range sm.States {
		statesString += "\n\t\t\t" + key + ":"
		statesString += "\n\t\t\t\t" + state.String()
	}
	if len(sm.States) > 0 {
		statesString += "\t\t]\n"
	} else {
		statesString += "]\n"
	}

	return fmt.Sprintf(`{
		Name: %s
		Comment: %s
		StartAt: %s
		Version: %s
		States: %s}`,
		sm.Name,
		sm.Comment,
		sm.StartAt,
		sm.Version,
		statesString)
}

// GetFunctionNames retrieves all functions defined in the StateMachine, and duplicates are allowed
func (sm *StateMachine) GetFunctionNames() []string {
	funcs := make([]string, 0)
	for _, v := range sm.States {
		res, ok := v.(HasResources)
		if ok {
			funcName := res.GetResources()
			funcs = append(funcs, funcName...)
		}
	}

	return funcs
}

// TODO: delete this functions when you're done with choice
/*func dagBuilding(funcs []*function.Function, currentState State, builder *fc.DagBuilder, condBuilder *fc.ChoiceBranchBuilder, stateMachine *StateMachine) (*fc.Dag, []*function.Function, error) {
	switch currentState.GetType() {
	case "Task":
		return nil, nil, nil
	case "Choice":
		// gets the conditions from matches
		//conds := make([]fc.Condition, 0)
		//for _, c := range currentState.Choices {
		//	conds = append(conds, c.Operation)
		//}
		//if condBuilder == nil {
		//	condBuilder = builder.AddChoiceNode(conds...)
		//}
		//
		//i := 0
		//for condBuilder.HasNextBranch() {
		//	nextState := stateMachine.States[currentState.Choices[i].Next]
		//	subDag, _, errSub := dagBuilding(funcs, nextState, NewDagBuilder(), condBuilder, stateMachine)
		//	if errSub != nil {
		//		return nil, nil, errSub
		//	}
		//	// assign the function (Task state) to the next branch
		//	condBuilder.NextBranch(subDag, nil)
		//	i++
		//}
		// dag, err := condBuilder.EndChoiceAndBuild()
		return nil, nil, nil
	case "Fail":
		return nil, nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported task type: %s", currentState.GetType())
	}
}*/

func (sm *StateMachine) Equals(comparer types.Comparable) bool {
	sm2 := comparer.(*StateMachine)

	if len(sm.States) != len(sm2.States) {
		return false
	}
	// checks if all states are equal
	for k := range sm.States {
		if !sm.States[k].Equals(sm2.States[k]) {
			return false
		}
	}
	return sm.StartAt == sm2.StartAt &&
		sm.Name == sm2.Name &&
		// sm.Comment == sm2.Comment &&
		sm.Version == sm2.Version
}
