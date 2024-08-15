package asl

import "github.com/grussorusso/serverledge/internal/types"

type PassState struct {
	Type StateType
}

func (p *PassState) Equals(cmp types.Comparable) bool {
	p2 := cmp.(*PassState)
	return p.Type == p2.Type
}

func NewEmptyPass() *PassState {
	return &PassState{
		Type: Pass,
	}
}

func (p *PassState) ParseFrom(jsonData []byte) (State, error) {
	//TODO implement me
	panic("implement me")
}

func (p *PassState) GetNext() []State {
	//TODO implement me
	panic("implement me")
}

func (p *PassState) GetType() StateType {
	return Pass
}

func (p *PassState) String() string {
	return "Pass"
}
