package asl

type WaitState struct{}

func (w *WaitState) GetNext() []State {
	//TODO implement me
	panic("implement me")
}

func (w *WaitState) GetType() StateType {
	return Wait
}
