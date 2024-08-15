package asl

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
	"github.com/grussorusso/serverledge/utils"
)

type TaskState struct {
	Type             StateType       // Necessary
	Resource         Path            // Necessary
	Next             string          // Necessary when End = false
	Parameters       PayloadTemplate // Optional
	ResultSelector   PayloadTemplate // Optional
	Retry            *Retry          // Optional
	Catch            *Catch          // Optional
	TimeoutSeconds   uint32          // Optional, in seconds
	HeartbeatSeconds uint32          // Optional, in seconds. Smaller than timeoutSeconds
	End              bool            // default false
}

func (t *TaskState) ParseFrom(jsonData []byte) (State, error) {

	var err error
	resource, err := utils.JsonExtract(jsonData, "Resource")
	if err != nil {
		return nil, fmt.Errorf("resource field is mandatory for a task state, but it is not defined")
	}

	resourcePath, err := NewPath(resource)
	if err != nil {
		return nil, err
	}
	t.Resource = resourcePath

	t.End = utils.JsonExtractBool(jsonData, "End")
	if !t.End {
		jsonNext, errTask := utils.JsonExtract(jsonData, "Next")
		if errTask != nil {
			return nil, fmt.Errorf("next field is mandatory for a non-terminal task state, but it is not defined")
		}
		t.Next = jsonNext
	}

	t.Parameters.json = utils.JsonExtractStringOrDefault(jsonData, "Parameters", "")
	t.ResultSelector.json = utils.JsonExtractStringOrDefault(jsonData, "ResultSelector", "")
	t.Retry = utils.JsonExtractObjectOrDefault(jsonData, "Retry", &Retry{}).(*Retry)
	t.Catch = utils.JsonExtractObjectOrDefault(jsonData, "Catch", &Catch{}).(*Catch)
	t.TimeoutSeconds = uint32(utils.JsonExtractIntOrDefault(jsonData, "TimeoutSeconds", 0))
	t.HeartbeatSeconds = uint32(utils.JsonExtractIntOrDefault(jsonData, "HeartbeatSeconds", 0))

	if t.HeartbeatSeconds > t.TimeoutSeconds {
		return nil, fmt.Errorf("HeartbeatSeconds %d exceeds timeout %d", t.HeartbeatSeconds, t.TimeoutSeconds)
	}

	return t, nil
}

func NewTerminalTask(resource Path) *TaskState {
	return &TaskState{
		Type:             Task,
		Resource:         resource,
		Next:             "",
		Parameters:       PayloadTemplate{""},
		ResultSelector:   PayloadTemplate{""},
		Retry:            NoRetry(),
		Catch:            NoCatch(),
		TimeoutSeconds:   0,
		HeartbeatSeconds: 0,
		End:              true,
	}
}

func NewEmptyTask() *TaskState {
	return &TaskState{
		Type:             Task,
		Resource:         "",
		Next:             "",
		Parameters:       PayloadTemplate{""},
		ResultSelector:   PayloadTemplate{""},
		Retry:            nil,
		Catch:            nil,
		TimeoutSeconds:   0,
		HeartbeatSeconds: 0,
		End:              true,
	}
}

func (t *TaskState) GetNext() []State {
	//TODO implement me
	panic("implement me")
}

func (t *TaskState) GetType() StateType {
	return Task
}

func (t *TaskState) Equals(cmp types.Comparable) bool {
	t2 := cmp.(*TaskState)
	return t.Type == t2.Type &&
		t.Resource == t2.Resource &&
		t.Next == t2.Next &&
		t.Parameters.json == t2.Parameters.json &&
		t.ResultSelector.json == t2.ResultSelector.json &&
		t.Retry.Equals(t2.Retry) &&
		t.Catch.Equals(t2.Catch) &&
		t.TimeoutSeconds == t2.TimeoutSeconds &&
		t.HeartbeatSeconds == t2.HeartbeatSeconds &&
		t.End == t2.End
}

func (t *TaskState) String() string {
	str := fmt.Sprint("{",
		"\n\t\t\t\tType: ", t.GetType(),
		"\n\t\t\t\tResource: ", t.Resource,
		"\n")

	if t.Next != "" {
		str += fmt.Sprintf(`\t\t\t\tNext: %s\n`, t.Next)
	}

	if t.Parameters.json != "" {
		str += fmt.Sprintf("\t\t\t\tParameters: %s\n", t.Parameters.json)
	}

	if t.ResultSelector.json != "" {
		str += fmt.Sprint("\t\t\t\tResultSelector: ", t.ResultSelector)
	}
	if t.Retry != nil {
		str += fmt.Sprintf("\t\t\t\tRetry: %v\n", t.Retry)
	}
	if t.Catch != nil {
		str += fmt.Sprintf("\t\t\t\tCatch: %v\n", t.Catch)
	}
	if t.TimeoutSeconds != 0 {
		str += fmt.Sprintf("\t\t\t\tTimeoutSeconds: %d\n", t.TimeoutSeconds)
	}
	if t.HeartbeatSeconds != 0 {
		str += fmt.Sprintf("\t\t\t\tHeartbeatSeconds: %d\n", t.HeartbeatSeconds)
	}
	if t.End != false {
		str += fmt.Sprintf("\t\t\t\tEnd: %v\n", t.End)
	}
	str += "\t\t\t}\n"
	return str
}
