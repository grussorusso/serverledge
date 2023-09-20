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

type HasBranch interface {
	setBranchId(number int)
	GetBranchId() int
}

type Display interface {
	ToString() string
	GetId() string
	Name() string
}

type Executable interface {
	// Exec defines the execution of the Dag. TODO: The output are saved in the struct, or returned?
	Exec() (map[string]interface{}, error)
}

type HasInput interface {
	// AddInput adds an input node, if compatible. For some DagNodes can be called multiple times
	AddInput(dagNode DagNode) error
}

type HasOutput interface {
	// AddOutput  adds a result node, if compatible. For some DagNodes can be called multiple times
	AddOutput(dagNode DagNode) error
}

type ReceivesInput interface {
	// ReceiveInput gets the input and if necessary tries to convert into a suitable representation for the executing function
	ReceiveInput(input map[string]interface{}) error
}

type ReceivesOutput interface {
	// PrepareOutput maps the outputMap of the current node to the inputMap of the next nodes
	PrepareOutput(output map[string]interface{}) error
}

type HasNext interface {
	GetNext() []DagNode
}
