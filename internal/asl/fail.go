package asl

import "github.com/grussorusso/serverledge/internal/types"

type FailState struct {
	Type StateType
	// Error is a error identifier to be used in a retry State
	Error     string
	ErrorPath Path
	// Cause is a human-readable message
	Cause     string
	CausePath Path
}

func (f *FailState) Validate(stateNames []string) error {
	//TODO implement me
	panic("implement me")
}

func (f *FailState) IsEndState() bool {
	return true
}

func (f *FailState) Equals(cmp types.Comparable) bool {
	f2 := cmp.(*FailState)
	return f.Type == f2.Type &&
		f.Error == f2.Error &&
		f.ErrorPath == f2.ErrorPath &&
		f.Cause == f2.Cause &&
		f.CausePath == f2.CausePath
}

func NewEmptyFail() *FailState {
	return &FailState{
		Type: Fail,
	}
}

func (f *FailState) ParseFrom(jsonData []byte) (State, error) {
	//TODO implement me
	panic("implement me")
}

func (f *FailState) GetType() StateType {
	return Fail
}

func (f *FailState) String() string {
	return "Fail"
}
