package fc

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
	"github.com/lithammer/shortuuid"
	"time"
)

type MergeMode int

const (
	AddNewMapEntry  = iota // The output type will be a map of key-values
	AddToArrayEntry        // The output type will be a map with a single array of values, with repetition
	AddToSetEntry          // The output type will be a map with a single array of unique values
)

// FanInNode receives and merges multiple input and produces a single result
type FanInNode struct {
	Id          string
	InputFrom   []map[string]interface{} // we need this because fan in should know which node to wait.
	OutputTo    DagNode
	FanInDegree int
	timeout     time.Duration
	Mode        MergeMode
	input       map[string]interface{}
}

var DefaultTimeout = 60 * time.Second

func NewFanInNode(mergeMode MergeMode, nillableTimeout *time.Duration) *FanInNode {
	timeout := nillableTimeout
	if timeout == nil {
		timeout = &DefaultTimeout
	}

	return &FanInNode{
		Id:          shortuuid.New(),
		InputFrom:   nil,
		OutputTo:    nil,
		FanInDegree: 0,
		timeout:     *timeout,
		Mode:        mergeMode,
	}
}

func (f *FanInNode) Equals(cmp types.Comparable) bool {
	switch cmp.(type) {
	case *FanInNode:
		f2 := cmp.(*FanInNode)
		//for i := 0; i < len(f.InputFrom); i++ {
		//	if f.InputFrom[i] != f2.InputFrom[i] {
		//		return false
		//	}
		//}
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
	//if len(f.InputFrom) == f.FanInDegree {
	//	return errors.New("input already present in node")
	//}
	//
	//f.InputFrom = append(f.InputFrom, dagNode)
	return nil
}

func (f *FanInNode) AddOutput(dagNode DagNode) error {
	//if f.OutputTo != nil {
	//	return errors.New("result already present in node")
	//}

	f.OutputTo = dagNode
	return nil
}

func (f *FanInNode) ReceiveInput(input map[string]interface{}) error {
	// TODO: devi ricevere gli input separatamente dai nodi precedenti.
	f.input = input
	return nil
}

func (f *FanInNode) PrepareOutput(output map[string]interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (f *FanInNode) GetNext() []DagNode {
	// we only have one output
	// TODO: we should wait for function to complete!
	arr := make([]DagNode, 1)
	if f.OutputTo == nil {
		panic("you forgot to initialize OutputTo for FanInNode")
	}
	arr[0] = f.OutputTo
	return arr
}

func (f *FanInNode) Width() int {
	return f.FanInDegree
}

func (f *FanInNode) Name() string {
	return "Fan In"
}

func (f *FanInNode) ToString() string {
	return fmt.Sprintf("[FanInNode(%d)]", f.FanInDegree)
}
