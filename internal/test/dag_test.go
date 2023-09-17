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

	dag, err := fc.CreateSequenceDag(fArr...)
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

	dag, err := fc.CreateChoiceDag(func() (*fc.Dag, error) { return fc.CreateSequenceDag(fArr...) }, arr...)
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

func TestChoiceDag_BuiltWithNextBranch(t *testing.T) {
	m := make(map[string]interface{})
	m["input"] = 1

	f, _, err := initializeSameFunctionSlice(1, "py")
	u.AssertNil(t, err)

	dag, err := fc.NewDagBuilder().
		AddChoiceNode(
			fc.NewConstCondition(false),
			fc.NewSmallerCondition(2, 1),
			fc.NewConstCondition(true),
		).
		NextBranch(fc.CreateSequenceDag(f)).
		NextBranch(fc.CreateSequenceDag(f)).
		NextBranch(fc.CreateSequenceDag(f)).
		EndChoiceAndBuild()

	width := len(dag.Start.Next.(*fc.ChoiceNode).Alternatives)

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

// TestBroadcastDag verifies that a broadcast dag is created correctly with fan out, simple nodes and fan in.
// All dag branches have the same sequence of simple nodes.
func TestBroadcastDag(t *testing.T) {
	m := make(map[string]interface{}) // TODO: stai testando questo
	m["input"] = 1
	width := 3
	f, fArr, err := initializeSameFunctionSlice(width, "js")
	u.AssertNil(t, err)

	fmt.Println("==== Parallel Dag ====")
	dag, errDag := fc.CreateBroadcastDag(func() (*fc.Dag, error) { return fc.CreateSequenceDag(fArr...) }, width)
	u.AssertNil(t, errDag)
	dag.Print()

	u.AssertNonNil(t, dag.Start)
	u.AssertNonNil(t, dag.End)
	u.AssertEquals(t, width, dag.Width)
	u.AssertNonNil(t, dag.Nodes)
	u.AssertEquals(t, width*width+2, len(dag.Nodes)) // 1 (fanOut) + 1 (fanIn) + width * width (simpleNodes)

	dagNodes := fc.NewNodeSetFrom(dag.Nodes)

	u.AssertTrue(t, dagNodes.Contains(dag.Start.Next))

	for _, n := range dag.Nodes {
		switch n.(type) {
		case *fc.FanOutNode:
			fanOut := n.(*fc.FanOutNode)
			u.AssertEquals(t, len(fanOut.OutputTo), fanOut.FanOutDegree)
			u.AssertEquals(t, width, fanOut.FanOutDegree)
			for _, s := range fanOut.OutputTo {
				u.AssertTrue(t, dagNodes.Contains(s))
			}
		case *fc.FanInNode:
			fanIn := n.(*fc.FanInNode)
			u.AssertEquals(t, width, fanIn.FanInDegree)
			u.AssertEquals(t, dag.End, fanIn.OutputTo)
		case *fc.SimpleNode:
			u.AssertTrue(t, n.(*fc.SimpleNode).Func == f.Name)
		default:
			t.FailNow()
		}
	}
}

func TestScatterDag(t *testing.T) {
	f, err := initializeExamplePyFunction()
	u.AssertNil(t, err)
	width := 3
	dag, errDag := fc.CreateScatterSingleFunctionDag(f, width)
	u.AssertNil(t, errDag)
	dag.Print()

	u.AssertNonNil(t, dag.Start)
	u.AssertNonNil(t, dag.End)
	u.AssertEquals(t, dag.Width, width) // width is fixed at dag definition-time
	u.AssertNonNil(t, dag.Nodes)
	u.AssertEquals(t, width+2, len(dag.Nodes)) // 1 (fanOut) + 1 (fanIn) + width (simpleNodes)

	dagNodes := fc.NewNodeSetFrom(dag.Nodes)
	u.AssertTrue(t, dagNodes.Contains(dag.Start.Next))
	_, ok := dag.Start.Next.(*fc.FanOutNode)
	u.AssertTrue(t, ok)
	simpleNodeChainedToFanIn := 0
	for _, n := range dag.Nodes {
		switch node := n.(type) {
		case *fc.FanOutNode:
			fanOut := node
			u.AssertEquals(t, len(fanOut.OutputTo), fanOut.FanOutDegree)
			u.AssertEquals(t, width, fanOut.FanOutDegree)
			for _, s := range fanOut.OutputTo {
				u.AssertTrue(t, dagNodes.Contains(s))
			}
		case *fc.FanInNode:
			fanIn := node
			u.AssertEquals(t, width, fanIn.FanInDegree)
			u.AssertEquals(t, dag.End, fanIn.OutputTo)
		case *fc.SimpleNode:
			u.AssertTrue(t, n.(*fc.SimpleNode).Func == f.Name)
			_, chainedToFanIn := node.OutputTo.(*fc.FanInNode)
			u.AssertTrue(t, chainedToFanIn)
			simpleNodeChainedToFanIn++
		default:
			t.FailNow() // there aren't other node in this dag
		}
	}
	u.AssertEquals(t, width, simpleNodeChainedToFanIn)
}

func TestCreateBroadcastMultiFunctionDag(t *testing.T) {
	width1 := 2
	f, fArrPy, err := initializeSameFunctionSlice(width1, "py")
	u.AssertNil(t, err)
	width2 := 3
	_, fArrJs, err2 := initializeSameFunctionSlice(width2, "js")
	u.AssertNil(t, err2)
	dag, errDag := fc.CreateBroadcastMultiFunctionDag(
		func() (*fc.Dag, error) { return fc.CreateSequenceDag(fArrPy...) },
		func() (*fc.Dag, error) { return fc.CreateSequenceDag(fArrJs...) },
	)
	u.AssertNil(t, errDag)
	dag.Print()

	u.AssertNonNil(t, dag.Start)
	u.AssertNonNil(t, dag.End)
	u.AssertEquals(t, 2, dag.Width)
	u.AssertNonNil(t, dag.Nodes)
	u.AssertEquals(t, width1+width2+2, len(dag.Nodes)) // 1 (fanOut) + 1 (fanIn) + width (simpleNodes)

	dagNodes := fc.NewNodeSetFrom(dag.Nodes)
	u.AssertTrue(t, dagNodes.Contains(dag.Start.Next))
	_, ok := dag.Start.Next.(*fc.FanOutNode)
	u.AssertTrue(t, ok)
	// TODO: test that there are simple nodes chained to fan out
	// TODO: test that all simple nodes are chained to a fan in node.
	simpleNodeChainedToFanIn := 0
	for _, n := range dag.Nodes {
		switch node := n.(type) {
		case *fc.FanOutNode:
			fanOut := node
			u.AssertEquals(t, len(fanOut.OutputTo), fanOut.FanOutDegree)
			for _, s := range fanOut.OutputTo {
				u.AssertTrue(t, dagNodes.Contains(s))
			}
		case *fc.FanInNode:
			fanIn := node
			u.AssertEquals(t, dag.Width, fanIn.FanInDegree)
			u.AssertEquals(t, dag.End, fanIn.OutputTo)
		case *fc.SimpleNode:
			u.AssertTrue(t, node.Func == f.Name)
			if _, ok := node.OutputTo.(*fc.FanInNode); ok {
				simpleNodeChainedToFanIn++
			}
		default:
			continue
		}
	}
	u.AssertEquals(t, 2, simpleNodeChainedToFanIn)
}

// TestDagBuilder tests a complex Dag with every type of node in it
//
//		    [Start ]
//	           |
//	        [Simple]
//	 	       |
//		[====Choice====] // 1 == 4, 1 != 4
//	       |        |
//	    [Simple] [FanOut] // scatter
//	       |       |3|
//	       |     [Simple]
//	       |       |3|
//	       |     [FanIn ] // AddToArrayEntry
//	       |        |
//	       |---->[ End  ]
func TestDagBuilder(t *testing.T) {
	// t.Skip() // FIXME: Infinite loop!!
	f, err := initializeExamplePyFunction()
	u.AssertNil(t, err)
	width := 3
	dag, err := fc.NewDagBuilder().
		AddSimpleNode(f).
		AddChoiceNode(fc.NewEqCondition(1, 4), fc.NewDiffCondition(1, 4)).
		NextBranch(fc.CreateSequenceDag(f)).
		NextBranch(fc.NewDagBuilder().
			AddScatterFanOutNode(width).
			ForEachParallelBranch(func() (*fc.Dag, error) { return fc.CreateSequenceDag(f) }).
			AddFanInNode(fc.AddToArrayEntry).
			Build()).
		EndChoiceAndBuild()

	u.AssertNil(t, err)
	dagNodes := fc.NewNodeSetFrom(dag.Nodes)
	simpleNodeChainedToFanIn := 0
	for _, n := range dag.Nodes {
		switch node := n.(type) {
		case *fc.FanOutNode:
			fanOut := node
			u.AssertEquals(t, len(fanOut.OutputTo), fanOut.FanOutDegree)
			u.AssertEquals(t, width, fanOut.FanOutDegree)
			for _, s := range fanOut.OutputTo {
				u.AssertTrue(t, dagNodes.Contains(s))
			}
		case *fc.FanInNode:
			fanIn := node
			u.AssertEquals(t, width, fanIn.FanInDegree)
			u.AssertEquals(t, dag.End, fanIn.OutputTo)
		case *fc.SimpleNode:
			u.AssertTrue(t, node.Func == f.Name)
			if _, ok := node.GetNext()[0].(*fc.FanInNode); ok {
				simpleNodeChainedToFanIn++
			}
		case *fc.ChoiceNode:
			choice := node
			u.AssertEquals(t, len(choice.Conditions), len(choice.Alternatives))

			// specific for this test
			firstAlternative := choice.Alternatives[0].(*fc.SimpleNode)
			secondAlternative := choice.Alternatives[1].(*fc.FanOutNode)

			u.AssertTrue(t, dagNodes.Contains(firstAlternative))
			u.AssertTrue(t, dagNodes.Contains(secondAlternative))
			u.AssertEquals(t, firstAlternative.OutputTo, dag.End)
			// checking fan out - simples - fan in
			for i := range secondAlternative.OutputTo {
				simple, ok := secondAlternative.OutputTo[i].(*fc.SimpleNode)
				u.AssertTrue(t, ok)
				_, okFanIn := simple.OutputTo.(*fc.FanInNode)
				u.AssertTrue(t, okFanIn)
			}

		default:
			t.FailNow()
		}
	}
	u.AssertEquals(t, 3, simpleNodeChainedToFanIn)
	dag.Print()
}
