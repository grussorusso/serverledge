package test

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/fc"
	u "github.com/grussorusso/serverledge/utils"
	"math/rand"
	"testing"
)

// test for dag connections
func TestEmptyDag(t *testing.T) {
	input := 1
	m := make(map[string]interface{})
	m["input"] = input
	dag, err := fc.CreateEmptyDag()
	u.AssertNil(t, err)

	dag.Print()

	u.AssertNonNil(t, dag.Start)
	u.AssertNonNil(t, dag.End)
	u.AssertEquals(t, dag.Width, 1)
	u.AssertNonNil(t, dag.Nodes)
	u.AssertEquals(t, dag.Start.Next, dag.End)
}

// TestSimpleDag creates a simple Dag with one StartNode, two SimpleNode and one EndNode, executes it and gets the result.
func TestSimpleDag(t *testing.T) {
	input := 1
	m := make(map[string]interface{})
	m["input"] = input
	length := 2

	f, fArr, err := initializeSameFunctionSlice(length, "js")
	u.AssertNil(t, err)

	dag, err := fc.CreateSequenceDag(fArr)
	u.AssertNil(t, err)
	dag.Print()

	u.AssertNonNil(t, dag.Start)
	u.AssertNonNil(t, dag.End)
	u.AssertEquals(t, dag.Width, 1)
	u.AssertNonNil(t, dag.Nodes)
	u.AssertEquals(t, len(dag.Nodes), length)
	for _, n := range dag.Nodes {
		u.AssertTrue(t, n.(*fc.SimpleNode).Func == f.Name)
	}
	dagNodes := fc.NewNodeSetFrom(dag.Nodes)

	u.AssertTrue(t, dagNodes.Contains(dag.Start.Next))

	if len(dag.Nodes) > 1 {
		u.AssertEquals(t, dag.Nodes[0].(*fc.SimpleNode).OutputTo, dag.Nodes[1])
	} else if len(dag.Nodes) == 1 {
		u.AssertEquals(t, dag.Nodes[0].(*fc.SimpleNode).OutputTo, dag.End)
	}
}

func TestChoiceDag(t *testing.T) {
	m := make(map[string]interface{})
	m["input"] = 1

	arr := make([]fc.Condition, 3)
	arr[0] = fc.NewConstCondition(false)
	arr[1] = fc.NewConstCondition(rand.Int()%2 == 0)
	arr[2] = fc.NewConstCondition(true)
	width := len(arr)
	f, fArr, err := initializeSameFunctionSlice(1, "js")
	u.AssertNil(t, err)

	dag, err := fc.CreateChoiceDag(arr, func() (*fc.Dag, error) { return fc.CreateSequenceDag(fArr) })
	u.AssertNil(t, err)
	fmt.Println("==== Choice  Dag ====")
	dag.Print()

	u.AssertNonNil(t, dag.Start)
	u.AssertNonNil(t, dag.End)
	u.AssertEquals(t, dag.Width, width)
	u.AssertNonNil(t, dag.Nodes)
	// u.AssertEquals(t, width+1, len(dag.Nodes))

	dagNodes := fc.NewNodeSetFrom(dag.Nodes)

	u.AssertTrue(t, dagNodes.Contains(dag.Start.Next))
	for i, n := range dag.Nodes {
		if i == 0 {
			choice := n.(*fc.ChoiceNode)
			u.AssertEquals(t, len(choice.Conditions), len(choice.Alternatives))
			for _, s := range choice.Alternatives {
				u.AssertTrue(t, dagNodes.Contains(s))
				u.AssertEquals(t, s.(*fc.SimpleNode).OutputTo, dag.End)
			}
		} else {
			u.AssertTrue(t, n.(*fc.SimpleNode).Func == f.Name)
		}
	}
}
func TestParallelDag(t *testing.T) {
	m := make(map[string]interface{})
	m["input"] = 1
	width := 6
	f, fArr, err := initializeSameFunctionSlice(width, "js")
	u.AssertNil(t, err)

	fmt.Println("==== Parallel Dag ====")
	dag := fc.CreateParallelDag(width, fArr)
	// dag := CreateParallelDag2(func() Dag { return CreateSequenceDag(fArr) })
	dag.Print()

	u.AssertNonNil(t, dag.Start)
	u.AssertNonNil(t, dag.End)
	u.AssertEquals(t, dag.Width, width)
	u.AssertNonNil(t, dag.Nodes)
	u.AssertEquals(t, len(dag.Nodes), width+2) // 1 (fanOut) + 1 (fanIn) + width (simpleNodes)

	dagNodes := fc.NewNodeSetFrom(dag.Nodes)

	u.AssertTrue(t, dagNodes.Contains(dag.Start.Next))

	for _, n := range dag.Nodes {
		switch n.(type) {
		case *fc.FanOutNode:
			fanOut := n.(*fc.FanOutNode)
			// u.AssertEquals(t, fanOut.InputFrom, dag.Start)
			u.AssertEquals(t, len(fanOut.OutputTo), fanOut.FanOutDegree)
			u.AssertEquals(t, width, fanOut.FanOutDegree)
			for _, s := range fanOut.OutputTo {
				u.AssertTrue(t, dagNodes.Contains(s))
				// u.AssertTrue(t, s.(*fc.SimpleNode).InputFrom == fanOut)
			}
		case *fc.FanInNode:
			fanIn := n.(*fc.FanInNode)
			u.AssertEquals(t, width, fanIn.FanInDegree)
			u.AssertEquals(t, dag.End, fanIn.OutputTo)
			for _, s := range fanIn.InputFrom {
				u.AssertTrue(t, dagNodes.Contains(s))
				u.AssertTrue(t, s.(*fc.SimpleNode).OutputTo == fanIn)
			}
		case *fc.SimpleNode:
			u.AssertTrue(t, n.(*fc.SimpleNode).Func == f.Name)
		default:
			continue
		}
	}
}

// TestDagBuilder TODO: tests a complex Dag with every type of node in it
func TestDagBuilder(t *testing.T) {
	f, err := initializeExamplePyFunction()
	u.AssertNil(t, err)
	dag, err := fc.NewDagBuilder().
		AddSimpleNode(f).
		AddSimpleNode(f).
		Build()

	u.AssertNil(t, err)
	dag.Print()
}
