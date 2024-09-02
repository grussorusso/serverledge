package fc

import (
	"github.com/grussorusso/serverledge/internal/types"
	"github.com/lithammer/shortuuid"
)

type PassNode struct {
	Id       DagNodeId
	NodeType DagNodeType
}

func NewPassNode(message string) *PassNode {
	passNode := PassNode{
		Id:       DagNodeId("pass_" + shortuuid.New()),
		NodeType: Pass,
	}
	return &passNode
}

func (p *PassNode) Equals(cmp types.Comparable) bool {
	//TODO implement me
	panic("implement me")
}

func (p *PassNode) String() string {
	//TODO implement me
	panic("implement me")
}

func (p *PassNode) GetId() DagNodeId {
	//TODO implement me
	panic("implement me")
}

func (p *PassNode) Name() string {
	//TODO implement me
	panic("implement me")
}

func (p *PassNode) Exec(compRequest *CompositionRequest, params ...map[string]interface{}) (map[string]interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (p *PassNode) AddOutput(dag *Dag, dagNode DagNodeId) error {
	//TODO implement me
	panic("implement me")
}

func (p *PassNode) CheckInput(input map[string]interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (p *PassNode) PrepareOutput(dag *Dag, output map[string]interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (p *PassNode) GetNext() []DagNodeId {
	//TODO implement me
	panic("implement me")
}

func (p *PassNode) Width() int {
	//TODO implement me
	panic("implement me")
}

func (p *PassNode) setBranchId(number int) {
	//TODO implement me
	panic("implement me")
}

func (p *PassNode) GetBranchId() int {
	//TODO implement me
	panic("implement me")
}

func (p *PassNode) GetNodeType() DagNodeType {
	//TODO implement me
	panic("implement me")
}
