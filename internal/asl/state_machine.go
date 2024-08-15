package asl

import (
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
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

func parseStates(statesData string) (map[string]State, error) {
	states := make(map[string]State)
	err := jsonparser.ObjectEach([]byte(statesData), func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		stateType, _, _, err2 := jsonparser.Get(value, "Type")
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
	sm, funcs, err := smBuilding(funcs, startingState, NewsmBuilder(), nil, stateMachine)
	if err != nil {
		return nil, err
	}

}*/

func (sm *StateMachine) getFunctions() []string {
	funcs := make([]string, 0)
	for _, v := range sm.States {
		res, ok := v.(HasResources)
		if ok {
			funcName := res.GetResource()
			funcs = append(funcs, funcName)
		}
	}

	return funcs
}

func (sm *StateMachine) ToFunctionComposition() (*fc.FunctionComposition, error) {
	dag, err := fc.NewDagBuilder().Build() //TODO: build dag
	if err != nil {
		return nil, fmt.Errorf("could not build sm: %v", err)
	}

	funcNames := sm.getFunctions()
	funcs := make([]*function.Function, 0)
	for _, f := range funcNames {
		funcObj, ok := function.GetFunction(f)
		if !ok {
			return nil, fmt.Errorf("function does not exists")
		}
		funcs = append(funcs, funcObj)
	}

	comp := fc.NewFC(sm.Name, *dag, funcs, false)
	return &comp, nil
}

func (sm *StateMachine) Equals(comparer types.Comparable) bool {
	sm2 := comparer.(*StateMachine)

	if len(sm.States) != len(sm2.States) {
		return false
	}

	for k := range sm.States {
		if !sm.States[k].Equals(sm2.States[k]) {
			return false
		}
	}
	return sm.StartAt == sm2.StartAt &&
		sm.Name == sm2.Name &&
		sm.Comment == sm2.Comment &&
		sm.Version == sm2.Version
}
