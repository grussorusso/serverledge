package asl

type SucceedState struct{}

func (s *SucceedState) GetNext() []State {
	//TODO implement me
	panic("implement me")
}

func (s *SucceedState) GetType() StateType {
	return Succeed
}
