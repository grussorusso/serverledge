package asl

// State is the common interface for ASL states. StateTypes are Task, Parallel, Map, Pass, Wait, Choice, Succeed, Fail
type State interface {
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

// Type is a string field that indicates the state type in a ASL state machine
type Type = StateType

// Comment is an optional field that is skipped by the parser
type Comment = string

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

// Retry is a field in Task, Parallel and Map states which retries the state for a period or for a specified number of times
type Retry struct {
	ErrorEquals     []string
	IntervalSeconds int
	BackoffRate     int
	MaxAttempts     int
}
type CanRetry interface {
	GetRetryOpt() Retry
}

// Catch is a field in Task, Parallel and Map states. When a state reports an error and either there is no Retrier, or retries have failed to resolve the error, the interpreter scans through the Catchers in array order, and when the Error Name appears in the value of a Catcher’s "ErrorEquals" field, transitions the machine to the state named in the value of the "Next" field. The reserved name "States.ALL" appearing in a Retrier’s "ErrorEquals" field is a wildcard and matches any Error Name.
type Catch struct {
	ErrorEquals []string
	ResultPath  string
	Next        string
}
type CanCatch interface {
	GetCatchOpt() Catch
}

type HasResources interface {
	GetResource() string
}
