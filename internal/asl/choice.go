package asl

import "github.com/grussorusso/serverledge/internal/fc"

type ChoiceState struct {
	matches []Match
}

func (c *ChoiceState) GetNext() State {
	//TODO implement me
	panic("implement me")
}

func (c *ChoiceState) GetType() int {
	//TODO implement me
	panic("implement me")
}

type Match struct {
	Variable  string
	Operation fc.Condition
	Next      string
}
