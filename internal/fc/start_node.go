package fc

import (
	"errors"
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
	"github.com/lithammer/shortuuid"
	"reflect"
)

// StartNode is a DagNode from which the execution of the Dag starts. Invokes the first DagNode
type StartNode struct {
	Id   string
	Next DagNode
}

func NewStartNode() *StartNode {
	return &StartNode{
		Id: shortuuid.New(),
	}
}

func (s *StartNode) Equals(cmp types.Comparable) bool {
	switch cmp.(type) {
	case *StartNode:
		return s.Next == cmp.(*StartNode).Next
	default:
		return false
	}
}

func (s *StartNode) AddInput(dagNode DagNode) error {
	panic("not supported")
}

func (s *StartNode) AddOutput(dagNode DagNode) error {

	switch dagNode.(type) {
	case *StartNode:
		return errors.New(fmt.Sprintf("you cannot add an result of type %s to a %s", reflect.TypeOf(dagNode), reflect.TypeOf(s)))
	default:
		s.Next = dagNode
	}
	return nil
}

func (s *StartNode) Exec() (map[string]interface{}, error) {
	panic("you can't exec a start node")
}

func (s *StartNode) ReceiveInput(input map[string]interface{}) error {
	panic("it's useless to receive input from a startNode. Just send it in output")
	// TODO: are you sure?
}

// PrepareOutput for StartNode just send to the next node what it receives
func (s *StartNode) PrepareOutput(output map[string]interface{}) error {
	err := s.Next.ReceiveInput(output)
	return err
}

func (s *StartNode) GetNext() []DagNode {
	// we only have one output
	arr := make([]DagNode, 1)
	if s.Next != nil {
		arr[0] = s.Next
		return arr
	}
	panic("you forgot to initialize next for StartNode")
}

func (s *StartNode) Width() int {
	return 1
}

func (s *StartNode) Name() string {
	return "Start "
}

func (s *StartNode) ToString() string {
	return fmt.Sprintf("[%s]-next->%s", s.Name(), s.Next.Name())
}

func (s *StartNode) setBranchId(number int) {
}
func (s *StartNode) GetBranchId() int {
	return 0
}

func (s *StartNode) GetId() string {
	return s.Id
}
