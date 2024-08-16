package fc

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/asl"
	"github.com/grussorusso/serverledge/internal/function"
)

// FromASL parses a AWS State Language specification file and returns a Function Composition with the corresponding Serverledge Dag
// The name of the composition should not be the file name by default, to avoid problems when adding the same composition multiple times.
func FromASL(name string, aslSrc []byte) (*FunctionComposition, error) {
	stateMachine, err := asl.ParseFrom(name, aslSrc)
	if err != nil {
		return nil, fmt.Errorf("could not parse the ASL file: %v", err)
	}
	return FromStateMachine(stateMachine, true)
}

/* ============== Build from ASL States =================== */

// BuildFromTaskState adds a SimpleNode to the previous Node. The simple node will have id as specified by the name parameter
func BuildFromTaskState(builder *DagBuilder, t *asl.TaskState, name string) (*DagBuilder, error) {
	f, found := function.GetFunction(t.Resource)
	if !found {
		return nil, fmt.Errorf("non existing function in composition: %s", t.Resource)
	}
	builder = builder.AddSimpleNodeWithId(f, name)
	fmt.Printf("Added simple node with f: %s\n", f.Name)
	return builder, nil
}

// BuildFromChoiceState adds a ChoiceNode as defined in the ChoiceState and connects it to the previous Node
func BuildFromChoiceState(builder *DagBuilder, c *asl.ChoiceState, name string) (*DagBuilder, error) {
	// TODO: implement me
	return builder, nil
}

// BuildFromParallelState adds a FanOutNode and a FanInNode and as many branches as defined in the ParallelState
func BuildFromParallelState(builder *DagBuilder, c *asl.ParallelState, name string) (*DagBuilder, error) {
	// TODO: implement me
	return builder, nil
}

// BuildFromMapState is not compatible with Serverledge at the moment
func BuildFromMapState(builder *DagBuilder, c *asl.MapState, name string) (*DagBuilder, error) {
	// TODO: implement me
	// TODO: implement MapNode
	panic("not compatible with serverledge currently")
	// return builder, nil
}

// BuildFromPassState adds a SimpleNode with an identity function
func BuildFromPassState(builder *DagBuilder, p *asl.PassState, name string) (*DagBuilder, error) {
	// TODO: implement me
	return builder, nil
}

// BuildFromWaitState adds a Simple node with a sleep function for the specified time as described in the WaitState
func BuildFromWaitState(builder *DagBuilder, w *asl.WaitState, name string) (*DagBuilder, error) {
	// TODO: implement me
	return builder, nil
}

// BuildFromSucceedState is not fully compatible with serverledge, but it adds an EndNode
func BuildFromSucceedState(builder *DagBuilder, s *asl.SucceedState, name string) (*DagBuilder, error) {
	// TODO: implement me
	return builder, nil
}

// BuildFromFailState is not fully compatible with serverledge, but it adds an EndNode
func BuildFromFailState(builder *DagBuilder, s *asl.FailState, name string) (*DagBuilder, error) {
	// TODO: implement me
	return builder, nil
}
