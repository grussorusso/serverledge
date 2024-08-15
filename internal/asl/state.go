package asl

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
)

// State is the common interface for ASL states. StateTypes are Task, Parallel, Map, Pass, Wait, Choice, Succeed, Fail
type State interface {
	fmt.Stringer
	types.Comparable
	GetType() StateType
}

// StateType for ASL states
type StateType string

const (
	Task     StateType = "Task"
	Parallel StateType = "Parallel"
	Map      StateType = "Map"
	Pass     StateType = "Pass"
	Wait     StateType = "Wait"
	Choice   StateType = "Choice"
	Succeed  StateType = "Succeed"
	Fail     StateType = "Fail"
)

// InputPath is a field that indicates from where to read input
type InputPath = string

// OutputPath is a field that indicates where to write output
type OutputPath = string

// HasNext is an interface for non-terminal states.
// Fail, Succeed and any other state with End=true are terminal states.
// Fail and Succeed should not implement this interface
type HasNext interface {
	// HasSingleNext return true for all states except Choice.
	HasSingleNext() bool
	// Next returns the next state, if exists. Otherwise returns an error. For the Choice state returns an error.
	Next() (string, error)
}

// CanEnd is an interface with a boolean method. If true, marks the state machine end. Valid for Task, Parallel, Map, Pass and Wait states
type CanEnd interface {
	IsEndState() bool
}

// ResultPath is a string field that indicates the result path
type ResultPath = string

// Parameters is a JSON string payload template with names and values of input parameters for Task, Map and Parallel states
type Parameters = string
type HasParameters interface {
	GetParameters() Parameters
}

// ResultSelector is a reference path that indicates where to place the output relative to the raw input.
type ResultSelector = string // TODO: what is that

type HasResources interface {
	GetResource() string
}

type Parseable interface {
	ParseFrom(jsonData []byte) (State, error)
}
