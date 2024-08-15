package asl

import "github.com/grussorusso/serverledge/internal/types"

type MapState struct {
	Type StateType
}

func (m *MapState) Equals(cmp types.Comparable) bool {
	m2 := cmp.(*MapState)
	return m.Type == m2.Type
}

func NewEmptyMap() *MapState {
	return &MapState{
		Type: Map,
	}
}

func (m *MapState) ParseFrom(jsonData []byte) (State, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MapState) GetNext() []State {
	//TODO implement me
	panic("implement me")
}

func (m *MapState) GetType() StateType {
	return Map
}

func (m *MapState) String() string {
	return "Map"
}
