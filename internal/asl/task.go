package asl

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
	"github.com/grussorusso/serverledge/utils"
)

type TaskState struct {
	Type                 StateType       // Necessary
	Resource             string          // Necessary
	Next                 string          // Necessary when End = false
	Parameters           PayloadTemplate // Optional
	ResultSelector       PayloadTemplate // Optional
	Retry                *Retry          // Optional
	Catch                *Catch          // Optional
	TimeoutSeconds       uint32          // Optional, in seconds
	TimeoutSecondsPath   Path            // Optional, in seconds
	HeartbeatSeconds     uint32          // Optional, in seconds. Smaller than timeoutSeconds
	HeartbeatSecondsPath Path            // Optional, in seconds. Smaller than timeoutSeconds
	End                  bool            // default false
}

func (t *TaskState) ParseFrom(jsonData []byte) (State, error) {

	var err error
	resource, err := utils.JsonExtract(jsonData, "Resource")
	if err != nil {
		return nil, fmt.Errorf("resource field is mandatory for a task state, but it is not defined")
	}
	t.Resource = resource

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
	timeoutSecondsPath, errT := NewReferencePath(utils.JsonExtractStringOrDefault(jsonData, "TimeoutSecondsPath", ""))
	if errT != nil {
		return nil, errT
	}
	t.TimeoutSecondsPath = timeoutSecondsPath
	t.HeartbeatSeconds = uint32(utils.JsonExtractIntOrDefault(jsonData, "HeartbeatSeconds", 0))
	heartbeatSecondsPath, errH := NewReferencePath(utils.JsonExtractStringOrDefault(jsonData, "HeartbeatSecondsPath", ""))
	if errH != nil {
		return nil, errH
	}
	t.HeartbeatSecondsPath = heartbeatSecondsPath
	if t.HeartbeatSeconds > t.TimeoutSeconds {
		return nil, fmt.Errorf("HeartbeatSeconds %d exceeds timeout %d", t.HeartbeatSeconds, t.TimeoutSeconds)
	}

	return t, nil
}

func NewTerminalTask(resource string) *TaskState {
	return &TaskState{
		Type:                 Task,
		Resource:             resource,
		Next:                 "",
		Parameters:           PayloadTemplate{""},
		ResultSelector:       PayloadTemplate{""},
		Retry:                NoRetry(),
		Catch:                NoCatch(),
		TimeoutSeconds:       0,
		TimeoutSecondsPath:   "",
		HeartbeatSeconds:     0,
		HeartbeatSecondsPath: "",
		End:                  true,
	}
}

func NewNonTerminalTask(resource string, next string) *TaskState {
	return &TaskState{
		Type:                 Task,
		Resource:             resource,
		Next:                 next,
		Parameters:           PayloadTemplate{""},
		ResultSelector:       PayloadTemplate{""},
		Retry:                NoRetry(),
		Catch:                NoCatch(),
		TimeoutSeconds:       0,
		TimeoutSecondsPath:   "",
		HeartbeatSeconds:     0,
		HeartbeatSecondsPath: "",
		End:                  false,
	}
}

func NewEmptyTask() *TaskState {
	return &TaskState{
		Type:                 Task,
		Resource:             "",
		Next:                 "",
		Parameters:           PayloadTemplate{""},
		ResultSelector:       PayloadTemplate{""},
		Retry:                nil,
		Catch:                nil,
		TimeoutSeconds:       0,
		TimeoutSecondsPath:   "",
		HeartbeatSeconds:     0,
		HeartbeatSecondsPath: "",
		End:                  true,
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
		t.TimeoutSecondsPath == t2.TimeoutSecondsPath &&
		t.HeartbeatSeconds == t2.HeartbeatSeconds &&
		t.HeartbeatSecondsPath == t2.HeartbeatSecondsPath &&
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
	if t.TimeoutSeconds != 0 {
		str += fmt.Sprintf("\t\t\t\tTimeoutSecondsPath: %s\n", t.TimeoutSecondsPath)
	}
	if t.HeartbeatSeconds != 0 {
		str += fmt.Sprintf("\t\t\t\tHeartbeatSeconds: %d\n", t.HeartbeatSeconds)
	}
	if t.HeartbeatSeconds != 0 {
		str += fmt.Sprintf("\t\t\t\tHeartbeatSecondsPath: %s\n", t.HeartbeatSecondsPath)
	}
	if t.End != false {
		str += fmt.Sprintf("\t\t\t\tEnd: %v\n", t.End)
	}
	str += "\t\t\t}\n"
	return str
}
