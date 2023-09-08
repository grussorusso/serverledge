package fc

import (
	"errors"
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
)

type MergeMode int

const (
	AddNewMapEntry  = iota // The output type will be a map of key-values
	AddToArrayEntry        // The output type will be a map with a single array of values, with repetition
	AddToSetEntry          // The output type will be a map with a single array of unique values
)

// FanInNode receives and merges multiple input and produces a single result
type FanInNode struct {
	InputFrom   []DagNode
	OutputTo    DagNode
	FanInDegree int
	timeout     int // default 60
	Mode        MergeMode
}

func (f *FanInNode) Equals(cmp types.Comparable) bool {
	switch cmp.(type) {
	case *FanInNode:
		f2 := cmp.(*FanInNode)
		for i := 0; i < len(f.InputFrom); i++ {
			if f.InputFrom[i] != f2.InputFrom[i] {
				return false
			}
		}
		return f.FanInDegree == f2.FanInDegree &&
			f.OutputTo == f2.OutputTo // && f.timeout == f2.timeout
	default:
		return false
	}
}

func (f *FanInNode) Exec() (map[string]interface{}, error) {
	//TODO You must wait all output from all InputFrom nodes
	// or you should return an error after a timeout expires
	panic("implement me")
}

func (f *FanInNode) AddInput(dagNode DagNode) error {
	if len(f.InputFrom) == f.FanInDegree {
		return errors.New("input already present in node")
	}

	f.InputFrom = append(f.InputFrom, dagNode)
	return nil
}

func (f *FanInNode) AddOutput(dagNode DagNode) error {
	if f.OutputTo != nil {
		return errors.New("result already present in node")
	}

	f.OutputTo = dagNode
	return nil
}

func (f *FanInNode) ReceiveInput(input map[string]interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (f *FanInNode) PrepareOutput(output map[string]interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (f *FanInNode) GetNext() []DagNode {
	// we only have one output
	// TODO: we should wait for function to complete!
	arr := make([]DagNode, 1)
	if f.OutputTo != nil {
		arr[0] = f.OutputTo
		return arr
	}
	panic("you forgot to initialize OutputTo for FanInNode")
}

func (f *FanInNode) Width() int {
	return f.FanInDegree
}

func (f *FanInNode) Name() string {
	return "Fan In"
}

func (f *FanInNode) ToString() string {
	inputs := "["
	for i, inputFrom := range f.InputFrom {
		inputs += inputFrom.Name()
		if i != len(f.InputFrom)-1 {
			inputs += ", "
		}
	}
	inputs += "]"
	return fmt.Sprintf("[FanInNode(%d)]<-%s ", f.FanInDegree, inputs)
}
