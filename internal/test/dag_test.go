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
	fc.BranchNumber = 0

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
	fc.BranchNumber = 0

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

	for i, node := range dag.Nodes {
		if i == len(dag.Nodes)-1 {
			u.AssertEquals(t, node.(*fc.SimpleNode).OutputTo, dag.End)
		} else {
			u.AssertEquals(t, node.(*fc.SimpleNode).OutputTo, dag.Nodes[1])
		}
		u.AssertEquals(t, node.(*fc.SimpleNode).BranchId, 0)
	}

}

func TestChoiceDag(t *testing.T) {
	fc.BranchNumber = 0

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
		u.AssertEquals(t, n.GetBranchId(), i)
	}
}

func TestChoiceDag_BuiltWithNextBranch(t *testing.T) {
	fc.BranchNumber = 0

	m := make(map[string]interface{})
	m["input"] = 1
	length := 2
	f, fArr, err := initializeSameFunctionSlice(length, "py")
	u.AssertNil(t, err)

	dag, err := fc.NewDagBuilder().
		AddChoiceNode(
			fc.NewConstCondition(false),
			fc.NewSmallerCondition(2, 1),
			fc.NewConstCondition(true),
		).
		NextBranch(fc.CreateSequenceDag(fArr...)).
		NextBranch(fc.CreateSequenceDag(fArr...)).
		NextBranch(fc.CreateSequenceDag(fArr...)).
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
			u.AssertEquals(t, n.GetBranchId(), 0)
			u.AssertEquals(t, len(choice.Conditions), len(choice.Alternatives))
			for _, s := range choice.Alternatives {
				u.AssertTrue(t, dagNodes.Contains(s))
				if length == 1 {
					u.AssertEquals(t, s.(*fc.SimpleNode).OutputTo, dag.End)
				}
			}
		} else {
			u.AssertTrue(t, n.(*fc.SimpleNode).Func == f.Name)
			u.AssertTrue(t, n.GetBranchId() > 0)
			fmt.Println("branchId: ", n.GetBranchId())
		}

	}
}

// TestBroadcastDag verifies that a broadcast dag is created correctly with fan out, simple nodes and fan in.
// All dag branches have the same sequence of simple nodes.
func TestBroadcastDag(t *testing.T) {
	fc.BranchNumber = 0

	m := make(map[string]interface{}) // TODO: stai testando questo
	m["input"] = 1
	width := 3
	length := 3
	f, fArr, err := initializeSameFunctionSlice(length, "js")
	u.AssertNil(t, err)

	dag, errDag := fc.CreateBroadcastDag(func() (*fc.Dag, error) { return fc.CreateSequenceDag(fArr...) }, width)
	u.AssertNil(t, errDag)
	dag.Print()

	u.AssertNonNil(t, dag.Start)
	u.AssertNonNil(t, dag.End)
	u.AssertEquals(t, width, dag.Width)
	u.AssertNonNil(t, dag.Nodes)
	u.AssertEquals(t, length*width+2, len(dag.Nodes)) // 1 (fanOut) + 1 (fanIn) + width * length (simpleNodes)

	dagNodes := fc.NewNodeSetFrom(dag.Nodes)

	u.AssertTrue(t, dagNodes.Contains(dag.Start.Next))

	for _, n := range dag.Nodes {
		switch n.(type) {
		case *fc.FanOutNode:
			fanOut := n.(*fc.FanOutNode)
			u.AssertEquals(t, len(fanOut.OutputTo), fanOut.FanOutDegree)
			u.AssertEquals(t, width, fanOut.FanOutDegree)
			for i, s := range fanOut.OutputTo {
				u.AssertTrue(t, dagNodes.Contains(s))
				u.AssertEquals(t, s.GetBranchId(), i+1)
			}
			u.AssertEquals(t, n.GetBranchId(), 0)
		case *fc.FanInNode:
			fanIn := n.(*fc.FanInNode)
			u.AssertEquals(t, width, fanIn.FanInDegree)
			u.AssertEquals(t, dag.End, fanIn.OutputTo)
			u.AssertEquals(t, n.GetBranchId(), 4)
		case *fc.SimpleNode:
			u.AssertTrue(t, n.(*fc.SimpleNode).Func == f.Name)
			u.AssertTrue(t, n.GetBranchId() > 0 && n.GetBranchId() < 4)
		default:
			t.FailNow()
		}
	}
}

func TestScatterDag(t *testing.T) {
	fc.BranchNumber = 0

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
			for j, s := range fanOut.OutputTo {
				u.AssertTrue(t, dagNodes.Contains(s))
				u.AssertEquals(t, s.GetBranchId(), j+1)
			}
			u.AssertEquals(t, node.GetBranchId(), 0)
		case *fc.FanInNode:
			fanIn := node
			u.AssertEquals(t, width, fanIn.FanInDegree)
			u.AssertEquals(t, dag.End, fanIn.OutputTo)
			u.AssertEquals(t, fanIn.GetBranchId(), fanIn.FanInDegree+1)
		case *fc.SimpleNode:
			u.AssertTrue(t, n.(*fc.SimpleNode).Func == f.Name)
			_, chainedToFanIn := node.OutputTo.(*fc.FanInNode)
			u.AssertTrue(t, chainedToFanIn)
			u.AssertTrue(t, n.GetBranchId() > 0 && n.GetBranchId() < 4)
			simpleNodeChainedToFanIn++
		default:
			t.FailNow() // there aren't other node in this dag
		}
	}
	u.AssertEquals(t, width, simpleNodeChainedToFanIn)
}

func TestCreateBroadcastMultiFunctionDag(t *testing.T) {
	fc.BranchNumber = 0

	length1 := 2
	f, fArrPy, err := initializeSameFunctionSlice(length1, "py")
	u.AssertNil(t, err)
	length2 := 3
	_, fArrJs, err2 := initializeSameFunctionSlice(length2, "js")
	u.AssertNil(t, err2)
	dag, errDag := fc.CreateBroadcastMultiFunctionDag(
		func() (*fc.Dag, error) { return fc.CreateSequenceDag(fArrPy...) },
		func() (*fc.Dag, error) { return fc.CreateSequenceDag(fArrJs...) },
	)
	u.AssertNil(t, errDag)
	dag.Print()

	fanOutDegree := dag.Start.Next.(*fc.FanOutNode).FanOutDegree

	u.AssertNonNil(t, dag.Start)
	u.AssertNonNil(t, dag.End)
	u.AssertEquals(t, 2, dag.Width)
	u.AssertNonNil(t, dag.Nodes)
	u.AssertEquals(t, length1+length2+2, len(dag.Nodes)) // 1 (fanOut) + 1 (fanIn) + width (simpleNodes)

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
			// test that there are simple nodes chained to fan out
			for j, s := range fanOut.OutputTo {
				u.AssertTrue(t, dagNodes.Contains(s))
				u.AssertEquals(t, s.GetBranchId(), j+1)
			}
			u.AssertEquals(t, fanOut.GetBranchId(), 0)
		case *fc.FanInNode:
			fanIn := node
			u.AssertEquals(t, dag.Width, fanIn.FanInDegree)
			u.AssertEquals(t, dag.End, fanIn.OutputTo)
			u.AssertEquals(t, fanIn.GetBranchId(), fanIn.FanInDegree+1)
		default:
			continue
		case *fc.SimpleNode:
			u.AssertTrue(t, node.Func == f.Name)
			u.AssertTrue(t, node.GetBranchId() > 0 && node.GetBranchId() < fanOutDegree+1)
			if _, ok := node.OutputTo.(*fc.FanInNode); ok {
				simpleNodeChainedToFanIn++
			}
		}
	}
	// test that the right number of simple nodes is chained to a fan in node.
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
	fc.BranchNumber = 0

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
			u.AssertEquals(t, 2, fanOut.GetBranchId())
			for _, s := range fanOut.OutputTo {
				u.AssertTrue(t, dagNodes.Contains(s))
			}
		case *fc.FanInNode:
			fanIn := node
			u.AssertEquals(t, width, fanIn.FanInDegree)
			u.AssertEquals(t, dag.End, fanIn.OutputTo)
			u.AssertEquals(t, 6, fanIn.GetBranchId())
		case *fc.SimpleNode:
			u.AssertTrue(t, node.Func == f.Name)
			if _, ok := node.GetNext()[0].(*fc.FanInNode); ok {
				simpleNodeChainedToFanIn++
				u.AssertTrue(t, node.GetBranchId() > 2 && node.GetBranchId() < 6) // the parallel branches of fan out node
			} else if _, ok2 := node.GetNext()[0].(*fc.ChoiceNode); ok2 {
				u.AssertEquals(t, node.GetBranchId(), 0) // the first simple node
			} else if _, ok3 := node.GetNext()[0].(*fc.EndNode); ok3 {
				u.AssertEquals(t, node.GetBranchId(), 1) // the first branch of choice node
			} else {
				u.AssertTrue(t, node.GetBranchId() > 2 && node.GetBranchId() < 6) // the parallel branches of fan out node
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
			u.AssertEquals(t, choice.GetBranchId(), 0)
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

func TestVisit(t *testing.T) {
	f, err := initializeExamplePyFunction()
	u.AssertNil(t, err)
	complexDag, err := fc.NewDagBuilder().
		AddSimpleNode(f).
		AddChoiceNode(fc.NewEqCondition(1, 4), fc.NewDiffCondition(1, 4)).
		NextBranch(fc.CreateSequenceDag(f)).
		NextBranch(fc.NewDagBuilder().
			AddScatterFanOutNode(3).
			ForEachParallelBranch(func() (*fc.Dag, error) { return fc.CreateSequenceDag(f) }).
			AddFanInNode(fc.AddToArrayEntry).
			Build()).
		EndChoiceAndBuild()
	u.AssertNil(t, err)

	choice := complexDag.Start.Next.GetNext()[0]

	nodeList := make([]fc.DagNode, 0)
	visitedNodes := fc.VisitDag(complexDag.Start, nodeList, false)
	u.AssertEquals(t, len(complexDag.Nodes)+2, len(visitedNodes))

	visitedNodes = fc.VisitDag(complexDag.Start, nodeList, true)
	u.AssertEquals(t, len(complexDag.Nodes)+1, len(visitedNodes))

	visitedNodes = fc.VisitDag(choice, nodeList, false)
	u.AssertEquals(t, 8, len(visitedNodes))

	visitedNodes = fc.VisitDag(choice, nodeList, true)
	u.AssertEquals(t, 7, len(visitedNodes))

}
