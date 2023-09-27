package fc

import (
	"github.com/grussorusso/serverledge/internal/types"
)

// DagNode is an interface for a single node in the Dag
// all implementors must be pointers to a struct
type DagNode interface {
	types.Comparable
	Display
	Executable
	HasInput
	HasOutput
	ReceivesInput
	ReceivesOutput
	HasNext
	Width() int
	HasBranch
}

// HasBranch is a counter that represent the branch of a node in the dag.
// For a sequence dag, the branch is always 0.
// For a dag with a single choice node, the choice node has branch 0, the N alternatives have branch 1,2,...,N
// For a parallel dag with one fanOut and fanIn, the fanOut has branch 0, fanOut branches have branch 1,2,...,N and FanIn has branch N+1
type HasBranch interface {
	setBranchId(number int)
	GetBranchId() int
}

type Display interface {
	ToString() string
	GetId() DagNodeId
	Name() string
}

type Executable interface {
	// Exec defines the execution of the Dag. TODO: The output are saved in the struct, or returned?
	Exec(progress *Progress) (map[string]interface{}, error)
}

type HasInput interface {
	// AddInput adds an input node, if compatible. For some DagNodes can be called multiple times
	AddInput(dagNode DagNode) error
}

type HasOutput interface {
	// AddOutput  adds a result node, if compatible. For some DagNodes can be called multiple times
	AddOutput(dag *Dag, dagNode DagNodeId) error
}

type ReceivesInput interface {
	// ReceiveInput gets the input and if necessary tries to convert into a suitable representation for the executing function
	ReceiveInput(input map[string]interface{}) error
}

type ReceivesOutput interface {
	// PrepareOutput maps the outputMap of the current node to the inputMap of the next nodes
	PrepareOutput(dag *Dag, output map[string]interface{}) error
}

type HasNext interface {
	GetNext() []DagNodeId
}

func Equals[D DagNode](d1 D, d2 D) bool {
	return d1.Equals(d2)
}

//type UnpackDagNode struct {
//	DagNode DagNode
//}
//
//func (u *UnpackDagNode) UnmarshalJSON(b []byte) error {
//
//	startNode := StartNode{}
//	simpleNode := SimpleNode{}
//	choiceNode := ChoiceNode{}
//	fanOutNode := FanOutNode{}
//	fanInNode := FanInNode{}
//	endNode := EndNode{}
//
//	// startNode
//	err := json.Unmarshal(b, &startNode)
//
//	// no error, but we also need to make sure we unmarshaled something
//	if err == nil && smth1.Thing != "" {
//		u.Data = smth1
//		return nil
//	}
//
//	// abort if we have an error other than the wrong type
//	if _, ok := err.(*json.UnmarshalTypeError); err != nil && !ok {
//		return err
//	}
//
//	smth2 := &Something2{}
//	err = json.Unmarshal(b, smth2)
//	if err != nil {
//		return err
//	}
//
//	u.DagNode = smth2
//	return nil
//}
