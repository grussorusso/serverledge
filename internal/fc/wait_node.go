package fc

import (
	"github.com/grussorusso/serverledge/internal/types"
	"github.com/lithammer/shortuuid"
)

type WaitNode struct {
	Id             DagNodeId
	NodeType       DagNodeType
	DurationMillis int
}

func NewWaitNode(durationMillis int) *WaitNode {
	if durationMillis < 0 {
		durationMillis = 0
	}
	waitNode := WaitNode{
		Id:             DagNodeId("wait_" + shortuuid.New()),
		NodeType:       Wait,
		DurationMillis: durationMillis,
	}
	return &waitNode
}

func (w *WaitNode) Equals(cmp types.Comparable) bool {
	//TODO implement me
	panic("implement me")
}

func (w *WaitNode) String() string {
	//TODO implement me
	panic("implement me")
}

func (w *WaitNode) GetId() DagNodeId {
	//TODO implement me
	panic("implement me")
}

func (w *WaitNode) Name() string {
	//TODO implement me
	panic("implement me")
}

func (w *WaitNode) Exec(compRequest *CompositionRequest, params ...map[string]interface{}) (map[string]interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (w *WaitNode) AddOutput(dag *Dag, dagNode DagNodeId) error {
	//TODO implement me
	panic("implement me")
}

func (w *WaitNode) CheckInput(input map[string]interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (w *WaitNode) PrepareOutput(dag *Dag, output map[string]interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (w *WaitNode) GetNext() []DagNodeId {
	//TODO implement me
	panic("implement me")
}

func (w *WaitNode) Width() int {
	//TODO implement me
	panic("implement me")
}

func (w *WaitNode) setBranchId(number int) {
	//TODO implement me
	panic("implement me")
}

func (w *WaitNode) GetBranchId() int {
	//TODO implement me
	panic("implement me")
}

func (w *WaitNode) GetNodeType() DagNodeType {
	//TODO implement me
	panic("implement me")
}
