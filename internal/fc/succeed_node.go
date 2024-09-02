package fc

import (
	"github.com/grussorusso/serverledge/internal/types"
	"github.com/lithammer/shortuuid"
)

type SucceedNode struct {
	Id       DagNodeId
	NodeType DagNodeType
	Message  string
}

func NewSucceedNode(message string) *SucceedNode {
	succeeedNode := SucceedNode{
		Id:       DagNodeId("succeed_" + shortuuid.New()),
		NodeType: Succeed,
		Message:  message,
	}
	return &succeeedNode
}

func (s *SucceedNode) Equals(cmp types.Comparable) bool {
	//TODO implement me
	panic("implement me")
}

func (s *SucceedNode) String() string {
	//TODO implement me
	panic("implement me")
}

func (s *SucceedNode) GetId() DagNodeId {
	//TODO implement me
	panic("implement me")
}

func (s *SucceedNode) Name() string {
	//TODO implement me
	panic("implement me")
}

func (s *SucceedNode) Exec(compRequest *CompositionRequest, params ...map[string]interface{}) (map[string]interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (s *SucceedNode) AddOutput(dag *Dag, dagNode DagNodeId) error {
	//TODO implement me
	panic("implement me")
}

func (s *SucceedNode) CheckInput(input map[string]interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (s *SucceedNode) PrepareOutput(dag *Dag, output map[string]interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (s *SucceedNode) GetNext() []DagNodeId {
	//TODO implement me
	panic("implement me")
}

func (s *SucceedNode) Width() int {
	//TODO implement me
	panic("implement me")
}

func (s *SucceedNode) setBranchId(number int) {
	//TODO implement me
	panic("implement me")
}

func (s *SucceedNode) GetBranchId() int {
	//TODO implement me
	panic("implement me")
}

func (s *SucceedNode) GetNodeType() DagNodeType {
	//TODO implement me
	panic("implement me")
}
