package fc

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
	"github.com/lithammer/shortuuid"
)

type FailNode struct {
	Id       DagNodeId
	NodeType DagNodeType
	Error    string
	Cause    string

	/* (Serverledge specific) */

	// OutputTo for a SucceedNode is used to send the output to the EndNode
	OutputTo DagNodeId
	BranchId int
}

func NewFailNode(error, cause string) *FailNode {
	if len(error) > 20 {
		fmt.Printf("error string identifier should be less than 20 characters but is %d characters long\n", len(error))
	}
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

func (f *FailNode) Equals(cmp types.Comparable) bool {
	f2, ok := cmp.(*FailNode)
	if !ok {
		return false
	}
	return f.Id == f2.Id && f.NodeType == f2.NodeType && f.Error == f2.Error && f.Cause == f2.Cause
}

func (f *FailNode) AddOutput(dag *Dag, dagNode DagNodeId) error {
	_, ok := dag.Nodes[dagNode].(*EndNode)
	if !ok {
		return fmt.Errorf("the FailNode can only be chained to an end node")
	}
	f.OutputTo = dagNode
	return nil
}

//func (f *FailNode) PrepareOutput(dag *Dag, output map[string]interface{}) error {
//	return nil
//}

func (f *FailNode) GetNext() []DagNodeId {
	return []DagNodeId{f.OutputTo}
}

func (f *FailNode) Width() int {
	return 1
}

func (f *FailNode) Name() string {
	return " Fail "
}

func (f *FailNode) String() string {
	return fmt.Sprintf("[Fail: %s]", f.Error)
}

func (f *FailNode) GetId() DagNodeId {
	return f.Id
}

func (f *FailNode) setBranchId(number int) {
	f.BranchId = number
}

func (f *FailNode) GetBranchId() int {
	return f.BranchId
}

func (f *FailNode) GetNodeType() DagNodeType {
	return f.NodeType
}
