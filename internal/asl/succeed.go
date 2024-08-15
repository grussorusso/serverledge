package asl

import "github.com/grussorusso/serverledge/internal/types"

type SucceedState struct {
	Type StateType
}

func (s *SucceedState) Equals(cmp types.Comparable) bool {
	s2 := cmp.(*SucceedState)
	return s.Type == s2.Type
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

func (s *SucceedState) GetNext() []State {
	//TODO implement me
	panic("implement me")
}

func (s *SucceedState) GetType() StateType {
	return Succeed
}

func (s *SucceedState) String() string {
	return "Succeed"
}
