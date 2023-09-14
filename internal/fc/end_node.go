package fc

import (
	"errors"
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
	"reflect"
)

type Reason int

const (
	Success Reason = iota
	Failure
)

// EndNode is a DagNode that represents the end of the Dag.
type EndNode struct {
	// InputFrom []DagNode // TODO: maybe useless
	result map[string]interface{} // TODO: maybe useless
	Reason Reason                 // TODO: maybe useless
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

func (e *EndNode) Exec() (map[string]interface{}, error) {
	return e.result, nil
}

func (e *EndNode) AddInput(dagNode DagNode) error {
	switch dagNode.(type) {
	case *EndNode:
		return errors.New(fmt.Sprintf("you cannot add an input of type %s to a %s", reflect.TypeOf(dagNode), reflect.TypeOf(e)))
	default:
		return nil
	}
}

func (e *EndNode) AddOutput(dagNode DagNode) error {
	//TODO implement me
	panic("implement me")
}

func (e *EndNode) ReceiveInput(input map[string]interface{}) error {
	//if e.result != nil {
	//	return errors.New("input already received")
	//}
	e.result = input
	return nil
}

// PrepareOutput doesn't need to do nothing for EndNode
func (e *EndNode) PrepareOutput(output map[string]interface{}) error {
	// TODO: dovrebbe inviare il risultato o forse va bene che non fa nulla
	return nil
}

func (e *EndNode) GetNext() []DagNode {
	// we return an empty array, because this is the EndNode
	return make([]DagNode, 0)
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
