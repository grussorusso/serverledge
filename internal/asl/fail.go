package asl

import "github.com/grussorusso/serverledge/internal/types"

type FailState struct {
	Type StateType
}

func (f *FailState) Equals(cmp types.Comparable) bool {
	f2 := cmp.(*FailState)
	return f.Type == f2.Type
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
