package asl

type TaskState struct{}

func (t *TaskState) GetNext() []State {
	//TODO implement me
	panic("implement me")
}

func (t *TaskState) GetType() StateType {
	return Task
}
