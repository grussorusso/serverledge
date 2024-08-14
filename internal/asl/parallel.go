package asl

type ParallelState struct{}

func (p *ParallelState) GetNext() []State {
	//TODO implement me
	panic("implement me")
}

func (p *ParallelState) GetType() StateType {
	return Parallel
}
