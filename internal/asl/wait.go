package asl

import "github.com/grussorusso/serverledge/internal/types"

type WaitState struct {
	Type StateType
	Next string
	End  bool
}

func (w *WaitState) Equals(cmp types.Comparable) bool {
	w2 := cmp.(*WaitState)
	return w.Type == w2.Type
}

func NewEmptyWait() *WaitState {
	return &WaitState{
		Type: Wait,
	}
}

func (w *WaitState) String() string {
	//TODO implement me
	panic("implement me")
}

func (w *WaitState) ParseFrom(jsonData []byte) (State, error) {
	//TODO implement me
	panic("implement me")
}

func (w *WaitState) GetNext() (string, bool) {
	if w.End == false {
		return w.Next, true
	}
	return "", false
}

func (w *WaitState) GetType() StateType {
	return Wait
}
