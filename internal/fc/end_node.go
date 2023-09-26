package fc

import (
	"errors"
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
	"github.com/lithammer/shortuuid"
	"reflect"
)

type Reason int

const (
	Success Reason = iota
	Failure
)

// EndNode is a DagNode that represents the end of the Dag.
type EndNode struct {
	Id     DagNodeId
	Result map[string]interface{} // TODO: maybe useless
	// Reason Reason                 // TODO: maybe useless
}

func NewEndNode() *EndNode {
	return &EndNode{
		Id: DagNodeId(shortuuid.New()),
	}
}

func (e *EndNode) Equals(cmp types.Comparable) bool {
	switch cmp.(type) {
	case *EndNode:
		//for i := 0; i < len(e.InputFrom); i++ {
		//	if !e.InputFrom[i].Equals(cmp.(*EndNode).InputFrom[i]) {
		//		return false
		//	}
		//}
		return true
	default:
		return false
	}
}

func (e *EndNode) Exec(*Progress) (map[string]interface{}, error) {
	return e.Result, nil
}

func (e *EndNode) AddInput(dagNode DagNode) error {
	switch dagNode.(type) {
	case *EndNode:
		return errors.New(fmt.Sprintf("you cannot add an input of type %s to a %s", reflect.TypeOf(dagNode), reflect.TypeOf(e)))
	default:
		return nil
	}
}

func (e *EndNode) AddOutput(dag *Dag, dagNode DagNodeId) error {
	//TODO implement me
	panic("implement me")
}

func (e *EndNode) ReceiveInput(input map[string]interface{}) error {
	//if e.result != nil {
	//	return errors.New("input already received")
	//}
	e.Result = input
	return nil
}

// PrepareOutput doesn't need to do nothing for EndNode
func (e *EndNode) PrepareOutput(dag *Dag, output map[string]interface{}) error {
	// TODO: dovrebbe inviare il risultato o forse va bene che non fa nulla
	return nil
}

func (e *EndNode) GetNext() []DagNodeId {
	// we return an empty array, because this is the EndNode
	return make([]DagNodeId, 0)
}

func (e *EndNode) Width() int {
	return 1
}

func (e *EndNode) Name() string {
	return " End  "
}

func (e *EndNode) ToString() string {
	return fmt.Sprintf("[EndNode]")
}
func (e *EndNode) setBranchId(number int) {
}
func (e *EndNode) GetBranchId() int {
	return 0
}

func (e *EndNode) GetId() DagNodeId {
	return e.Id
}
