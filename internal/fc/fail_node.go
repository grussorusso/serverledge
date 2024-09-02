package fc

import (
	"github.com/grussorusso/serverledge/internal/types"
	"github.com/lithammer/shortuuid"
)

type FailNode struct {
	Id       DagNodeId
	NodeType DagNodeType
	Error    string
	Cause    string
}

func NewFailNode(error, cause string) *FailNode {
	fail := FailNode{
		Id:       DagNodeId("fail_" + shortuuid.New()),
		NodeType: Fail,
		Error:    error,
		Cause:    cause,
	}
	return &fail
}

func (f *FailNode) Exec(compRequest *CompositionRequest, params ...map[string]interface{}) (map[string]interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (f *FailNode) Width() int {
	//TODO implement me
	panic("implement me")
}

func (f *FailNode) Equals(cmp types.Comparable) bool {
	//TODO implement me
	panic("implement me")
}

func (f *FailNode) String() string {
	//TODO implement me
	panic("implement me")
}

func (f *FailNode) GetId() DagNodeId {
	//TODO implement me
	panic("implement me")
}

func (f *FailNode) Name() string {
	//TODO implement me
	panic("implement me")
}

func (f *FailNode) AddOutput(dag *Dag, dagNode DagNodeId) error {
	//TODO implement me
	panic("implement me")
}

func (f *FailNode) CheckInput(input map[string]interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (f *FailNode) PrepareOutput(dag *Dag, output map[string]interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (f *FailNode) GetNext() []DagNodeId {
	//TODO implement me
	panic("implement me")
}

func (f *FailNode) setBranchId(number int) {
	//TODO implement me
	panic("implement me")
}

func (f *FailNode) GetBranchId() int {
	//TODO implement me
	panic("implement me")
}

func (f *FailNode) GetNodeType() DagNodeType {
	//TODO implement me
	panic("implement me")
}
