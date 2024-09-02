package fc

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
	"github.com/lithammer/shortuuid"
)

type SucceedNode struct {
	Id         DagNodeId
	NodeType   DagNodeType
	InputPath  string
	OutputPath string

	/* (Serverledge specific) */

	// OutputTo for a SucceedNode is used to send the output to the EndNode
	OutputTo DagNodeId
	BranchId int
}

func NewSucceedNode(message string) *SucceedNode {
	succeedNode := SucceedNode{
		Id:       DagNodeId("succeed_" + shortuuid.New()),
		NodeType: Succeed,
	}
	return &succeedNode
}

func (s *SucceedNode) Exec(compRequest *CompositionRequest, params ...map[string]interface{}) (map[string]interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (s *SucceedNode) Equals(cmp types.Comparable) bool {
	s2, ok := cmp.(*SucceedNode)
	if !ok {
		return false
	}
	return s.Id == s2.Id && s.NodeType == s2.NodeType && s.InputPath == s2.InputPath && s.OutputPath == s2.OutputPath
}

func (s *SucceedNode) AddOutput(dag *Dag, dagNode DagNodeId) error {
	_, ok := dag.Nodes[dagNode].(*EndNode)
	if !ok {
		return fmt.Errorf("the SucceedNode can only be chained to an end node")
	}
	s.OutputTo = dagNode
	return nil
}

// PrepareOutput can be used in a SucceedNode to modify the composition output representation
func (s *SucceedNode) PrepareOutput(dag *Dag, output map[string]interface{}) error {
	if s.OutputPath != "" {
		return fmt.Errorf("OutputPath not currently implemented") // TODO: implement it
	}
	return nil
}

func (s *SucceedNode) GetNext() []DagNodeId {
	return []DagNodeId{s.OutputTo}
}

func (s *SucceedNode) Width() int {
	return 1
}

func (s *SucceedNode) Name() string {
	return "Success"
}

func (s *SucceedNode) String() string {
	return "[Succeed]"
}

func (s *SucceedNode) setBranchId(number int) {
	s.BranchId = number
}

func (s *SucceedNode) GetBranchId() int {
	return s.BranchId
}

func (s *SucceedNode) GetId() DagNodeId {
	return s.Id
}

func (s *SucceedNode) GetNodeType() DagNodeType {
	return s.NodeType
}
