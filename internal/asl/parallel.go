package asl

import "github.com/grussorusso/serverledge/internal/types"

type ParallelState struct {
	Type StateType
}

func (p *ParallelState) Equals(cmp types.Comparable) bool {
	p2 := cmp.(*ParallelState)
	return p.Type == p2.Type
}

func NewEmptyParallel() *ParallelState {
	return &ParallelState{
		Type: Parallel,
	}
}

func (p *ParallelState) ParseFrom(jsonData []byte) (State, error) {
	//TODO implement me
	panic("implement me")
}

func (p *ParallelState) GetNext() []State {
	//TODO implement me
	panic("implement me")
}

func (p *ParallelState) GetType() StateType {
	return Parallel
}

// FIXME: improve
func (p *ParallelState) String() string {
	return "Parallel"
}
