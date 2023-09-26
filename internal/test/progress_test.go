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

func complexProgress(t *testing.T, condition fc.Condition) (*fc.Progress, *fc.Dag) {
	py, err := initializeExamplePyFunction()
	u.AssertNil(t, err)

	notCondition := fc.NewPredicate().Not(condition).Build()

	dag, err := fc.NewDagBuilder().
		AddSimpleNode(py).
		AddChoiceNode(
			notCondition,
			condition,
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

	// Start node
	checkAndCompleteNext(t, progress, dag)

	// Simple Node 1
	checkAndCompleteNext(t, progress, dag)

	// Simple Node 2
	checkAndCompleteNext(t, progress, dag)

	// End node
	checkAndCompleteNext(t, progress, dag)

	// End of dag
	finishProgress(t, progress)
}

// TestProgressChoice1 tests the left branch
func TestProgressChoice1(t *testing.T) {
	condition := fc.NewPredicate().And(
		fc.NewEqCondition(1, 3),
		fc.NewGreaterCondition(1, 3),
	).Build()
	progress, dag := choiceProgress(t, condition)

	// Start node
	checkAndCompleteNext(t, progress, dag)

	// Choice node
	choice := checkAndCompleteNext(t, progress, dag).(*fc.ChoiceNode)

	// Simple node (left) // suppose the left condition is true
	checkAndCompleteChoice(t, progress, choice, dag)

	// End
	checkAndCompleteNext(t, progress, dag)

	// End of dag
	finishProgress(t, progress)

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
	checkAndCompleteNext(t, progress, dag)

	// Choice node
	choice := checkAndCompleteNext(t, progress, dag).(*fc.ChoiceNode)

	// Simple Node left is skipped, right is executed
	checkAndCompleteChoice(t, progress, choice, dag)

	// Simple Node right 2
	checkAndCompleteNext(t, progress, dag)

	// End node
	checkAndCompleteNext(t, progress, dag)

	// End of dag
	finishProgress(t, progress)

	progress.Print()
}

func TestParallelProgress(t *testing.T) {
	progress, dag := parallelProgress(t)

	// Start node
	checkAndCompleteNext(t, progress, dag)

	// FanOut node
	checkAndCompleteNext(t, progress, dag)

	// 3 Simple Nodes in parallel
	checkAndCompleteMultiple(t, progress, dag)
	// simpleNode1 := fanOut.GetNext()[0]
	// simpleNode2 := fanOut.GetNext()[1]
	// simpleNode3 := fanOut.GetNext()[2]

	// 2 Simple Nodes in parallel // here should get two nodes
	checkAndCompleteMultiple(t, progress, dag)
	// nextNode = progress.NextNodes()
	// simpleNodeCentral2 := simpleNode2.GetNext()[0]
	// u.AssertEquals(t, nextNode[0], simpleNodeCentral2.GetId())
	// u.AssertEquals(t, 3, progress.GetGroup(nextNode[0]))
	// err = progress.CompleteNode(nextNode[0])
	// u.AssertNil(t, err)
	// simpleNodeCentral3 := simpleNode3.GetNext()[0]
	// u.AssertEquals(t, nextNode[1], simpleNodeCentral3.GetId())
	// u.AssertEquals(t, 3, progress.GetGroup(nextNode[1]))
	// err = progress.CompleteNode(nextNode[1])
	// u.AssertNil(t, err)

	// 1 Simple node (parallel) right, bottom
	checkAndCompleteMultiple(t, progress, dag)
	// nextNode = progress.NextNodes()
	// simpleNodeBottom3 := simpleNodeCentral3.GetNext()[0]
	// u.AssertEquals(t, nextNode[0], simpleNodeBottom3.GetId())
	// u.AssertEquals(t, 4, progress.GetGroup(nextNode[0]))
	// err = progress.CompleteNode(nextNode[0])
	// u.AssertNil(t, err)

	// Fan in
	checkAndCompleteNext(t, progress, dag)

	// End node
	checkAndCompleteNext(t, progress, dag)

	// End of dag
	finishProgress(t, progress)

	progress.Print()
}

func TestComplexProgress(t *testing.T) {
	condition := fc.NewPredicate().And(
		fc.NewEqCondition(1, 3),
		fc.NewGreaterCondition(1, 3),
	).Build()
	progress, dag := complexProgress(t, condition)

	// Start node
	checkAndCompleteNext(t, progress, dag)

	// SimpleNode
	checkAndCompleteNext(t, progress, dag)

	// Choice
	choice := checkAndCompleteNext(t, progress, dag).(*fc.ChoiceNode)

	// Simple Node, FanOut
	checkAndCompleteChoice(t, progress, choice, dag)

	// End node
	checkAndCompleteNext(t, progress, dag)

	// End of dag
	finishProgress(t, progress)

	progress.Print()
}

func TestComplexProgress2(t *testing.T) {
	condition := fc.NewPredicate().And(
		fc.NewEqCondition(1, 1),
		fc.NewGreaterCondition(4, 3),
	).Build()
	progress, dag := complexProgress(t, condition)

	// Start node
	checkAndCompleteNext(t, progress, dag)

	// Simple Node
	checkAndCompleteNext(t, progress, dag)

	// Choice
	choice := checkAndCompleteNext(t, progress, dag).(*fc.ChoiceNode)

	// Simple Node, FanOut // suppose the fanout node at the right and all its children are skipped
	checkAndCompleteChoice(t, progress, choice, dag)

	// 3 Simple Nodes in parallel
	checkAndCompleteMultiple(t, progress, dag)

	// 3 other Simple Nodes
	checkAndCompleteMultiple(t, progress, dag)

	// Fan in
	checkAndCompleteNext(t, progress, dag)

	// End node
	checkAndCompleteNext(t, progress, dag)

	// End of dag
	finishProgress(t, progress)

	progress.Print()
}

func checkAndCompleteNext(t *testing.T, progress *fc.Progress, dag *fc.Dag) fc.DagNode {
	nextNode, err := progress.NextNodes()
	u.AssertNil(t, err)
	nodeId := nextNode[0]
	node, ok := dag.Find(nodeId)
	u.AssertTrue(t, ok)
	u.AssertEquals(t, nodeId, node.GetId())
	u.AssertEquals(t, progress.NextGroup, progress.GetGroup(nodeId))
	err = progress.CompleteNode(nodeId)
	u.AssertNil(t, err)
	return node
}

func checkAndCompleteChoice(t *testing.T, progress *fc.Progress, choice *fc.ChoiceNode, dag *fc.Dag) {
	nextNode, err := progress.NextNodes() // Simple1, Simple2
	u.AssertNil(t, err)
	simpleNodeLeft := choice.Alternatives[0]
	fanOut := choice.Alternatives[1]
	u.AssertEquals(t, nextNode[0], simpleNodeLeft)
	u.AssertEquals(t, nextNode[1], fanOut)
	u.AssertEquals(t, progress.NextGroup, progress.GetGroup(nextNode[0]))
	u.AssertEquals(t, progress.NextGroup, progress.GetGroup(nextNode[1]))

	_, _ = choice.Exec(nil)
	err = progress.CompleteNode(nextNode[choice.FirstMatch])
	u.AssertNil(t, err)
	nodeToSkip := choice.GetNodesToSkip(dag)
	err = progress.SkipAll(nodeToSkip)
	u.AssertNil(t, err)
}

func checkAndCompleteMultiple(t *testing.T, progress *fc.Progress, dag *fc.Dag) []fc.DagNode {
	nextNode, err := progress.NextNodes()
	completedNodes := make([]fc.DagNode, 0)
	u.AssertNil(t, err)
	for _, nodeId := range nextNode {
		node, ok := dag.Find(nodeId)
		u.AssertTrue(t, ok)
		u.AssertEquals(t, nodeId, node.GetId())
		u.AssertEquals(t, progress.NextGroup, progress.GetGroup(nodeId))
		err = progress.CompleteNode(nodeId)
		u.AssertNil(t, err)
		completedNodes = append(completedNodes, node)
	}
	return completedNodes
}

func finishProgress(t *testing.T, progress *fc.Progress) {
	nothing, err := progress.NextNodes()
	u.AssertNil(t, err)
	u.AssertEmptySlice(t, nothing)
}
