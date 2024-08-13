package asl

// State is the common interface for ASL states. StateTypes are Task, Parallel, Map, Pass, Wait, Choice, Succeed, Fail
type State interface {
	GetNext() State
	GetType() int
}

// StateType for ASL states
const (
	Task = iota
	Parallel
	Map
	Pass
	Wait
	Choice
	Succeed
	Fail
)
