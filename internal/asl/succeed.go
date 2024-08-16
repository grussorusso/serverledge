package asl

import "github.com/grussorusso/serverledge/internal/types"

type SucceedState struct {
	Type       StateType // Necessary
	InputPath  Path      // Optional, default $
	OutputPath Path      // Optional, default $
}

func (s *SucceedState) IsEndState() bool {
	return true
}

func (s *SucceedState) Equals(cmp types.Comparable) bool {
	s2 := cmp.(*SucceedState)
	return s.Type == s2.Type &&
		s.InputPath == s2.InputPath &&
		s.OutputPath == s2.OutputPath
}

func NewEmptySucceed() *SucceedState {
	return &SucceedState{
		Type: Succeed,
	}
}

func (s *SucceedState) ParseFrom(jsonData []byte) (State, error) {
	//TODO implement me
	panic("implement me")
}

func (s *SucceedState) GetType() StateType {
	return Succeed
}

func (s *SucceedState) String() string {
	return "Succeed"
}
