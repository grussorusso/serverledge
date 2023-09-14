package fc

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/function"
)

// DagBuilder is a utility struct that helps easily define the Dag, using the Builder pattern.
// Use NewDagBuilder() to safely initialize it. Then use the available methods to iteratively build the dag.
// Finally use Build() to get the complete Dag.
type DagBuilder struct {
	dag      Dag
	branches int
	prevNode DagNode
	errors   []error
}

type ChoiceBranchBuilder struct {
	dagBuilder DagBuilder
	completed  int
}

type ParallelBranchBuilder struct {
	dagBuilder DagBuilder
}

func NewDagBuilder() DagBuilder {
	db := DagBuilder{
		dag:      NewDAG(),
		branches: 1,
		errors:   make([]error, 0),
	}
	db.prevNode = db.dag.Start
	return db
}

// AddSimpleNode connects a simple node to the previous node
func (b DagBuilder) AddSimpleNode(f *function.Function) DagBuilder {
	simpleNode := NewSimpleNode(f.Name)
	err := b.dag.Chain(b.prevNode, simpleNode)
	if err != nil {
		b.errors = append(b.errors, err)
		return b
	}
	b.dag.AddNode(simpleNode)
	b.prevNode = simpleNode
	fmt.Println("Added simple node to Dag")
	return b
}

// AddChoiceNode connects a choice node to the previous node. From the choice node, multiple branch are created and each one of those must be fully defined
func (b DagBuilder) AddChoiceNode(conditions []Condition) ChoiceBranchBuilder {
	nErrors := len(b.errors)
	if nErrors > 0 {
		fmt.Printf("NextBranch skipped, because of %d error(s) in dagBuilder\n", nErrors)
		return ChoiceBranchBuilder{dagBuilder: b, completed: 0}
	}

	fmt.Println("Added choice node to Dag")
	choiceNode := NewChoiceNode(conditions)

	b.branches = len(conditions)
	err := b.dag.Chain(b.prevNode, choiceNode)
	if err != nil {
		b.errors = append(b.errors, err)
		return ChoiceBranchBuilder{dagBuilder: b, completed: 0}
	}
	b.dag.AddNode(choiceNode)
	b.prevNode = choiceNode
	b.dag.Width = len(conditions)
	emptyBranches := make([]DagNode, 0, b.branches)
	choiceNode.Alternatives = emptyBranches
	// we construct a new slice with capacity (b.branches) and size 0
	// Here we cannot chain directly, because we do not know which alternative to chain to which node
	// so we return a ChoiceBranchBuilder
	return ChoiceBranchBuilder{dagBuilder: b, completed: 0}
}

// AddFanOutNode connects a fanout node to the previous node. From the fanout node, multiple branch are created and each one of those must be fully defined, eventually ending in a FanInNode
func (b DagBuilder) AddFanOutNode(fanoutType FanOutType) ParallelBranchBuilder {
	nErrors := len(b.errors)
	if nErrors > 0 {
		fmt.Printf("NextBranch skipped, because of %d error(s) in dagBuilder\n", nErrors)
		return ParallelBranchBuilder{b}
	}
	fmt.Println("Added fan out node to Dag")
	return ParallelBranchBuilder{b}
}

// NextBranch is used to chain the next branch to a Dag and then returns the ChoiceBranchBuilder.
// Tip: use a NewDagBuilder() as a parameter, instead of manually creating the Dag!
// Internally, NextBranch replaces the StartNode of the input dag with the choice alternative
// and chains the last node of the dag to the EndNode of the building dag
func (c ChoiceBranchBuilder) NextBranch(dag Dag) ChoiceBranchBuilder {
	nErrors := len(c.dagBuilder.errors)
	if nErrors > 0 {
		fmt.Printf("NextBranch skipped, because of %d error(s) in dagBuilder\n", nErrors)
		return c
	}

	fmt.Println("Added simple node to a branch in choice node of Dag")
	if c.HasNextBranch() {
		// chains the alternative to the input dag, which is already connected to a whole series of nodes
		err := c.dagBuilder.dag.Chain(c.dagBuilder.prevNode.(*ChoiceNode).Alternatives[c.completed], dag.Start.Next)
		if err != nil {
			c.dagBuilder.errors = append(c.dagBuilder.errors, err)
			return c
		}
		// adds the nodes to the building dag
		for _, n := range dag.Nodes {
			c.dagBuilder.dag.AddNode(n)
			// chain the last node(s) of the input dag to the end node of the building dag
			if n.GetNext() != nil && len(n.GetNext()) > 0 && n.GetNext()[0] == dag.End {
				switch n.(type) {
				case *FanOutNode:
					errFanout := fmt.Errorf("you're trying to chain a fanout node to an end node. This will interrupt the execution immediately after the fanout is reached")
					c.dagBuilder.errors = append(c.dagBuilder.errors, errFanout)
					continue
				default:
					errEnd := c.dagBuilder.dag.ChainToEndNode(n)
					if errEnd != nil {
						c.dagBuilder.errors = append(c.dagBuilder.errors, errEnd)
						return c
					}
				}
			}
		}

		// so we completed a branch
		c.completed++
		c.dagBuilder.branches--
	} else {
		panic("There is not a NextBranch. Use EndChoiceAndBuild to end the choiceNode.")
	}
	return c
}

// EndNextBranch is used to chain the next choice branch to the end node of the dag
func (c ChoiceBranchBuilder) EndNextBranch() ChoiceBranchBuilder {
	nErrors := len(c.dagBuilder.errors)
	if nErrors > 0 {
		fmt.Printf("NextBranch skipped, because of %d error(s) in dagBuilder\n", nErrors)
		return c
	}

	if c.HasNextBranch() {
		fmt.Println("Ending branch for Dag")
		// chain the alternative of the choice node to the end node of the building dag
		err := c.dagBuilder.dag.ChainToEndNode(c.dagBuilder.prevNode.(*ChoiceNode).Alternatives[c.completed])
		if err != nil {
			c.dagBuilder.errors = append(c.dagBuilder.errors, err)
			return c
		}
		// ... and we completed a branch
		c.completed++
		c.dagBuilder.branches--
		if !c.HasNextBranch() {
			c.dagBuilder.prevNode = c.dagBuilder.dag.End
		}
	} else {
		fmt.Println("warning: Useless call EndNextBranch: all branch are ended")
	}
	return c
}

func (c ChoiceBranchBuilder) HasNextBranch() bool {
	return c.completed < c.dagBuilder.branches
}

// EndChoiceAndBuild connects all remaining branches to the end node and builds the dag
func (c ChoiceBranchBuilder) EndChoiceAndBuild() (*Dag, error) {
	for c.HasNextBranch() {
		c.EndNextBranch()
	}

	if len(c.dagBuilder.errors) > 0 {
		return nil, fmt.Errorf("build failed with errors:\n%v", c.dagBuilder.errors)
	}

	return &c.dagBuilder.dag, nil
}

// ForEach chains each (remaining) output of a choice node to the same subsequent node, then returns the DagBuilder
func (c ChoiceBranchBuilder) ForEach(dagger func() (*Dag, error)) ChoiceBranchBuilder {
	choiceNode := c.dagBuilder.prevNode.(*ChoiceNode)
	// we suppose the branches 0, ..., (completed-1) are already completed
	// once := true
	for i := c.completed; i < c.dagBuilder.branches; i++ {
		fmt.Printf("Adding dag to branch %d\n", i)
		// recreates a dag executing the same function
		dagCopy, errDag := dagger()
		if errDag != nil {
			c.dagBuilder.errors = append(c.dagBuilder.errors, errDag)
		}
		err := c.dagBuilder.dag.Chain(choiceNode, dagCopy.Start.Next)
		if err != nil {
			c.dagBuilder.errors = append(c.dagBuilder.errors, err)
		}
		// adds the nodes to the building dag, but only once!
		for _, n := range dagCopy.Nodes {
			c.dagBuilder.dag.AddNode(n)
			// chain the last node(s) of the input dag to the end node of the building dag
			if n.GetNext() != nil && len(n.GetNext()) > 0 && n.GetNext()[0] == dagCopy.End {
				switch n.(type) {
				case *FanOutNode:
					errFanout := fmt.Errorf("you're trying to chain a fanout node to an end node. This will interrupt the execution immediately after the fanout is reached")
					c.dagBuilder.errors = append(c.dagBuilder.errors, errFanout)
					continue
				default:
					errEnd := c.dagBuilder.dag.ChainToEndNode(n)
					if errEnd != nil {
						c.dagBuilder.errors = append(c.dagBuilder.errors, errEnd)
						return c
					}
				}
			}
		}
		// so we completed a branch
		c.completed++
	}
	return c
}

func (p ParallelBranchBuilder) ForEach(dagger func() Dag) ParallelBranchBuilder {
	fmt.Println("Added dag for each fanout output in Dag")
	return p
}

func (p ParallelBranchBuilder) AddFanInNode(fanInMode MergeMode) DagBuilder {
	fmt.Println("Added fan in node after fanout in Dag")
	return p.dagBuilder
}

// Build ends the single branch with an EndNode. If there is more than one branch, it panics!
func (b DagBuilder) Build() (*Dag, error) {

	switch b.prevNode.(type) {
	case nil:
		return &b.dag, nil
	case *EndNode:
		return &b.dag, nil
	default:
		err := b.dag.ChainToEndNode(b.prevNode)
		if err != nil {
			return nil, fmt.Errorf("failed to chain to end node: %v", err)
		}
	}
	return &b.dag, nil
}

func CreateEmptyDag() (*Dag, error) {
	return NewDagBuilder().Build()
}

func CreateSequenceDag(funcs []*function.Function) (*Dag, error) {
	builder := NewDagBuilder()
	for _, f := range funcs {
		builder = builder.AddSimpleNode(f)
	}
	return builder.Build()
}

func CreateChoiceDag(condArr []Condition, dagger func() (*Dag, error)) (*Dag, error) {
	return NewDagBuilder().
		AddChoiceNode(condArr).
		ForEach(dagger).
		EndChoiceAndBuild()
}

func CreateParallelDag2(dagger func() Dag) (*Dag, error) {
	return NewDagBuilder().
		AddFanOutNode(Broadcast).
		ForEach(dagger).
		AddFanInNode(AddNewMapEntry).
		Build()
}
