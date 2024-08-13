package asl

// State is the common interface for ASL states. StateTypes are Task, Parallel, Map, Pass, Wait, Choice, Succeed, Fail
type State interface {
	GetNext() State
	GetType() int
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

// Next is a string field that points to the Next state
type Next = string

// End is a boolean field. If true, marks the state machine end
type End = bool

// ResultPath is a string field that indicates the result path
type ResultPath = string

// Parameters indicates the payload template with names and values of input parameters for Task, Map and Parallel states
type Parameters = string // TODO: maybe string not ok
// ResultSelector is a reference path that indicates where to place the output relative to the raw input.
type ResultSelector = string // TODO: what is that

// Retry is a field in Task, Parallel and Map states which retries the state for a period or for a specified number of times
type Retry struct {
	ErrorEquals     []string
	IntervalSeconds int
	BackoffRate     int
	MaxAttempts     int
}

// Catch is a field in Task, Parallel and Map states. When a state reports an error and either there is no Retrier, or retries have failed to resolve the error, the interpreter scans through the Catchers in array order, and when the Error Name appears in the value of a Catcher’s "ErrorEquals" field, transitions the machine to the state named in the value of the "Next" field. The reserved name "States.ALL" appearing in a Retrier’s "ErrorEquals" field is a wildcard and matches any Error Name.
type Catch struct {
	ErrorEquals []string
	ResultPath  string
	Next        string
}
