package asl

type FailState struct{}

func (f *FailState) GetType() StateType {
	return Fail
}
