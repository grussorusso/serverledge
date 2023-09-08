package fc

import (
	"errors"
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
	"math"
	"strings"
)

// FanOutNode is a DagNode that receives one input and sends multiple result, produced in parallel
type FanOutNode struct {
	Id string
	// InputFrom    DagNode
	OutputTo     []DagNode
	FanOutDegree int
	Type         FanOutType
}
type FanOutType int

const (
	Broadcast = iota
	Scatter
)

func (f *FanOutNode) Equals(cmp types.Comparable) bool {
	switch cmp.(type) {
	case *FanOutNode:
		f2 := cmp.(*FanOutNode)
		for i := 0; i < len(f.OutputTo); i++ {
			if f.OutputTo[i] != f2.OutputTo[i] {
				return false
			}
		}
		return f.FanOutDegree == f2.FanOutDegree // &&
		// f.InputFrom == f2.InputFrom
	default:
		return false
	}
}

func (f *FanOutNode) Exec() (map[string]interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (f *FanOutNode) AddInput(dagNode DagNode) error {
	//if f.InputFrom != nil {
	//	return errors.New("input already present in node")
	//}
	//
	//f.InputFrom = dagNode
	return nil
}

func (f *FanOutNode) AddOutput(dagNode DagNode) error {
	if len(f.OutputTo) == f.FanOutDegree {
		return errors.New("cannot add more output. Create a FanOutNode with a higher fanout degree")
	}
	f.OutputTo = append(f.OutputTo, dagNode)
	return nil
}

func (f *FanOutNode) ReceiveInput(input map[string]interface{}) error {

	//TODO implement me
	panic("implement me")
}

func (f *FanOutNode) PrepareOutput(output map[string]interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (f *FanOutNode) GetNext() []DagNode {
	// we have multiple outputs
	if f.FanOutDegree <= 1 {
		panic("You should have used a SimpleNode or EndNode for fanOutDegree less than 2")
	}

	if f.FanOutDegree != len(f.OutputTo) {
		panic("The fanOutDegree and number of outputs does not match")
	}

	if f.OutputTo != nil {
		return f.OutputTo
	}
	panic("you forgot to initialize OutputTo for FanOutNode")
}

func (f *FanOutNode) Width() int {
	return f.FanOutDegree
}

func (f *FanOutNode) Name() string {
	n := f.FanOutDegree
	if n%2 == 0 {
		return strings.Repeat("-", 4*(n-1)-n/2) + "FanOut" + strings.Repeat("-", 3*(n-1)+n/2)
	} else {
		pad := "-------"
		return strings.Repeat(pad, int(math.Max(float64(n/2), 0.))) + "FanOut" + strings.Repeat(pad, int(math.Max(float64(n/2), 0.)))
	}
}

func (f *FanOutNode) ToString() string {
	outputs := ""
	for i, outputTo := range f.OutputTo {
		outputs += outputTo.Name()
		if i != len(f.OutputTo)-1 {
			outputs += ", "
		}
	}
	outputs += "]"
	return fmt.Sprintf("[FanOutNode(%d)]->%s ", f.FanOutDegree, outputs)
}
