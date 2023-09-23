package test

import (
	"github.com/grussorusso/serverledge/internal/fc"
	u "github.com/grussorusso/serverledge/utils"
	"testing"
)

func simpleProgress(t *testing.T) (*fc.Progress, *fc.Dag) {
	py, err := initializeExamplePyFunction()
	u.AssertNil(t, err)
	dag, err := fc.CreateSequenceDag(py, py)
	u.AssertNil(t, err)
	return fc.InitProgressRecursive("simple", dag), dag
}

func choiceProgress(t *testing.T, condition fc.Condition) (*fc.Progress, *fc.Dag) {
	py, err := initializeExamplePyFunction()
	u.AssertNil(t, err)

	notCondition := fc.NewPredicate().Not(condition).Build()

	dag, err := fc.NewDagBuilder().
		AddChoiceNode(
			notCondition,
			condition,
		).
		NextBranch(fc.CreateSequenceDag(py)).
		NextBranch(fc.CreateSequenceDag(py, py)).
		EndChoiceAndBuild()
	u.AssertNil(t, err)

	return fc.InitProgressRecursive("abc", dag), dag
}

func parallelProgress(t *testing.T) (*fc.Progress, *fc.Dag) {
	py, err := initializeExamplePyFunction()
	u.AssertNil(t, err)

	dag, err := fc.NewDagBuilder().
		AddBroadcastFanOutNode(3).
		NextFanOutBranch(fc.CreateSequenceDag(py)).
		NextFanOutBranch(fc.CreateSequenceDag(py, py)).
		NextFanOutBranch(fc.CreateSequenceDag(py, py, py)).
		AddFanInNode(fc.AddNewMapEntry).
		Build()
	u.AssertNil(t, err)

	return fc.InitProgressRecursive("abc", dag), dag
}

func complexProgress(t *testing.T) (*fc.Progress, *fc.Dag) {
	py, err := initializeExamplePyFunction()
	u.AssertNil(t, err)

	condition := fc.NewPredicate().And(
		fc.NewEqCondition(1, 3),
		fc.NewGreaterCondition(1, 3),
	).Build()
	notCondition := fc.NewPredicate().Not(condition).Build()

	dag, err := fc.NewDagBuilder().
		AddSimpleNode(py).
		AddChoiceNode(
			condition,
			notCondition,
		).
		NextBranch(fc.CreateSequenceDag(py)).
		NextBranch(fc.NewDagBuilder().
			AddBroadcastFanOutNode(3).
			ForEachParallelBranch(func() (*fc.Dag, error) { return fc.CreateSequenceDag(py, py) }).
			AddFanInNode(fc.AddNewMapEntry).
			Build()).
		EndChoiceAndBuild()
	u.AssertNil(t, err)

	return fc.InitProgressRecursive("abc", dag), dag
}

// TestProgressSequence tests a sequence dag with 2 simple node
func TestProgressSequence(t *testing.T) {
	progress, dag := simpleProgress(t)
	nextNode := progress.NextNodes()
	u.AssertEquals(t, 1, len(nextNode))
	u.AssertEquals(t, len(dag.Nodes)+2, len(progress.DagNodes)) // start + 2 simple + end

	// Start node
	u.AssertEquals(t, nextNode[0], dag.Start.GetId())
	u.AssertEquals(t, 0, progress.GetGroup(nextNode[0]))
	err := progress.CompleteNode(nextNode[0]) // completes start
	u.AssertNil(t, err)

	// Simple Node 1
	nextNode = progress.NextNodes()
	u.AssertEquals(t, nextNode[0], dag.Start.Next.GetId())
	u.AssertEquals(t, 1, progress.GetGroup(nextNode[0]))
	err = progress.CompleteNode(nextNode[0]) // completes simple 1
	u.AssertNil(t, err)

	// Simple Node 2
	nextNode = progress.NextNodes()
	u.AssertEquals(t, nextNode[0], dag.Start.Next.GetNext()[0].GetId())
	u.AssertEquals(t, 2, progress.GetGroup(nextNode[0]))
	err = progress.CompleteNode(nextNode[0]) // completes simple 2
	u.AssertNil(t, err)

	// End node
	nextNode = progress.NextNodes()
	u.AssertEquals(t, nextNode[0], dag.End.GetId())
	u.AssertEquals(t, 3, progress.GetGroup(nextNode[0]))
	err = progress.CompleteNode(nextNode[0]) // completes end
	u.AssertNil(t, err)

	// End of dag
	nothing := progress.NextNodes()
	u.AssertEmptySlice(t, nothing)
}

// TestProgressChoice1 tests the left branch
func TestProgressChoice1(t *testing.T) {
	condition := fc.NewPredicate().And(
		fc.NewEqCondition(1, 3),
		fc.NewGreaterCondition(1, 3),
	).Build()
	progress, dag := choiceProgress(t, condition)
	nextNode := progress.NextNodes()
	u.AssertEquals(t, 1, len(nextNode))
	u.AssertEquals(t, len(dag.Nodes)+2, len(progress.DagNodes)) // start + choice + 3 simple + end

	// Start node
	start := dag.Start
	u.AssertEquals(t, nextNode[0], start.GetId())
	u.AssertEquals(t, 0, progress.GetGroup(nextNode[0]))
	err := progress.CompleteNode(nextNode[0])
	u.AssertNil(t, err)

	// Choice node
	nextNode = progress.NextNodes()
	choice := start.Next.(*fc.ChoiceNode)
	u.AssertEquals(t, nextNode[0], choice.GetId())
	u.AssertEquals(t, 1, progress.GetGroup(nextNode[0]))
	err = progress.CompleteNode(nextNode[0])
	u.AssertNil(t, err)

	// Simple node (left) // suppose the left condition is true
	nextNode = progress.NextNodes() // Simple1, Simple2
	simpleNodeLeft := choice.Alternatives[0]
	simpleNodeRight1 := choice.Alternatives[1]
	u.AssertEquals(t, nextNode[0], simpleNodeLeft.GetId())
	u.AssertEquals(t, nextNode[1], simpleNodeRight1.GetId())
	u.AssertEquals(t, 2, progress.GetGroup(nextNode[0]))
	u.AssertEquals(t, 2, progress.GetGroup(nextNode[1]))
	err = progress.CompleteNode(nextNode[0])
	u.AssertNil(t, err)

	_, _ = choice.Exec()
	nodeToSkip := choice.GetNodesToSkip()
	err = progress.SkipAll(nodeToSkip) // simpleNodeRight1
	u.AssertNil(t, err)

	// End
	nextNode = progress.NextNodes()
	u.AssertEquals(t, dag.End.GetId(), nextNode[0])
	u.AssertEquals(t, 4, progress.GetGroup(nextNode[0]))
	err = progress.CompleteNode(nextNode[0])
	u.AssertNil(t, err)

	// End of dag
	nothing := progress.NextNodes()
	u.AssertEmptySlice(t, nothing)

	progress.Print()
}

// TestProgressChoice2 tests the right branch
func TestProgressChoice2(t *testing.T) {
	condition := fc.NewPredicate().And(
		fc.NewEqCondition(1, 1),
		fc.NewGreaterCondition(5, 3),
	).Build()
	progress, dag := choiceProgress(t, condition)

	// Start node
	nextNode := progress.NextNodes()
	start := dag.Start
	err := progress.CompleteNode(nextNode[0])
	u.AssertNil(t, err)

	// Choice node
	nextNode = progress.NextNodes()
	choice := start.Next.(*fc.ChoiceNode)
	err = progress.CompleteNode(nextNode[0])
	u.AssertNil(t, err)

	// Simple Node left is skipped
	nextNode = progress.NextNodes() // Simple1, Simple2
	simpleNodeLeft := choice.Alternatives[0]
	// right is executed
	simpleNodeRight1 := choice.Alternatives[1]
	u.AssertEquals(t, nextNode[0], simpleNodeLeft.GetId())
	u.AssertEquals(t, nextNode[1], simpleNodeRight1.GetId())
	u.AssertEquals(t, 2, progress.GetGroup(nextNode[0]))
	u.AssertEquals(t, 2, progress.GetGroup(nextNode[1]))

	// skipping nodes
	_, _ = choice.Exec()
	err = progress.CompleteNode(nextNode[choice.FirstMatch])
	nodeToSkip := choice.GetNodesToSkip()
	err = progress.SkipAll(nodeToSkip)

	// Simple Node right 2
	nextNode = progress.NextNodes()
	simpleNodeRight2 := simpleNodeRight1.GetNext()[0].(*fc.SimpleNode)
	u.AssertEquals(t, nextNode[0], simpleNodeRight2.GetId())
	u.AssertEquals(t, 3, progress.GetGroup(nextNode[0]))
	err = progress.CompleteNode(nextNode[0]) // completes simple right 2
	u.AssertNil(t, err)

	// End node
	nextNode = progress.NextNodes()
	u.AssertEquals(t, nextNode[0], dag.End.GetId())
	u.AssertEquals(t, 4, progress.GetGroup(nextNode[0]))
	err = progress.CompleteNode(nextNode[0]) // completes end
	u.AssertNil(t, err)

	// End of dag
	nothing := progress.NextNodes()
	u.AssertEmptySlice(t, nothing)

	progress.Print()
}

func TestParallelProgress(t *testing.T) {
	progress, dag := parallelProgress(t)

	// Start node
	nextNode := progress.NextNodes()
	start := dag.Start
	u.AssertEquals(t, nextNode[0], start.GetId())
	u.AssertEquals(t, 0, progress.GetGroup(nextNode[0]))
	err := progress.CompleteNode(nextNode[0])
	u.AssertNil(t, err)

	// FanOut node
	nextNode = progress.NextNodes()
	fanOut := dag.Start.Next
	u.AssertEquals(t, nextNode[0], fanOut.GetId())
	u.AssertEquals(t, 1, progress.GetGroup(nextNode[0]))
	err = progress.CompleteNode(nextNode[0])
	u.AssertNil(t, err)

	progress.Print()
}
