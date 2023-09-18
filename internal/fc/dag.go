package fc

import (
	"errors"
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
	"math"
	"strings"
)

// used to send output from parallel nodes to fan in node or to the next node
var outputChannel = make(chan map[string]interface{})

// Dag is a Workflow to drive the execution of the function composition
type Dag struct {
	Start *StartNode // a single start must be added
	Nodes []DagNode
	End   *EndNode // a single endNode must be added
	Width int      // width is the max fanOut degree of the Dag
}

func NewDAG() Dag {
	dag := Dag{
		Start: NewStartNode(),
		End:   NewEndNode(),
		Nodes: []DagNode{},
		Width: 1,
	}
	return dag
}

// TODO: only the subsequent APIs should be public: NewDag, Print, GetUniqueDagFunctions, Equals
//  the remaining should be private after the builder APIs work well!!!

// addNode can be used to add a new node to the Dag. Does not chain anything, but updates Dag width
func (dag *Dag) addNode(node DagNode) {
	dag.Nodes = append(dag.Nodes, node)
	// updates width
	nodeWidth := node.Width()
	if nodeWidth > dag.Width {
		dag.Width = nodeWidth
	}
}

// TODO: maybe can be removed in favor of Chain
func (dag *Dag) SetStart(node1 DagNode) error {
	return dag.Start.AddOutput(node1)
}

// Chain can be used to connect the output of node1 to the node2
func (dag *Dag) Chain(node1 DagNode, node2 DagNode) error {
	return node1.AddOutput(node2)
}

// ChainToEndNode (node, i) can be used as a shorthand to Chain(node, dag.end[i]) to chain a node to a specific end node
func (dag *Dag) ChainToEndNode(node1 DagNode) error {
	return dag.Chain(node1, dag.End)
}

func (dag *Dag) Print() string {
	var currentNode DagNode = dag.Start
	result := ""

	// prints the StartNode
	if dag.Width == 1 {
		result += fmt.Sprintf("[%s]\n   |\n", currentNode.Name())
	} else if dag.Width%2 == 0 {
		result += fmt.Sprintf("%s[%s]\n%s|\n", strings.Repeat(" ", 7*dag.Width/2-3), currentNode.Name(), strings.Repeat(" ", 7*dag.Width/2))
	} else {
		result += fmt.Sprintf("%s[%s]\n%s|\n", strings.Repeat(" ", 7*int(math.Floor(float64(dag.Width/2)))), currentNode.Name(), strings.Repeat(" ", 7*int(math.Floor(float64(dag.Width/2)))+3))
	}

	currentNodes := currentNode.GetNext()
	doneNodes := NewNodeSet()

	for len(currentNodes) > 0 {
		result += "["
		var currentNodesToAdd []DagNode
		for i, node := range currentNodes {
			result += fmt.Sprintf("%s", node.Name())

			doneNodes.AddIfNotExists(node)

			if i != len(currentNodes)-1 {
				result += "|"
			}
			var addNodes []DagNode
			switch t := node.(type) {
			case *ChoiceNode:
				addNodes = t.Alternatives
			default:
				addNodes = node.GetNext()
			}
			currentNodesToAdd = append(currentNodesToAdd, addNodes...)

		}
		result += "]\n"
		currentNodes = currentNodesToAdd
		if len(currentNodes) > 0 {
			result += strings.Repeat("   |   ", len(currentNodes)) + "\n"
		}
	}
	fmt.Println(result)
	return result
}

// TODO: assicurarsi che si esegua in parallelo
// TODO: aggiungere lo stato di avanzamento: ad esempio su ETCD, compresi i parametri di input/output
func (dag *Dag) Execute(input map[string]interface{}) (map[string]interface{}, error) {

	if &dag.Start == nil && &dag.End == nil && dag.Width == 0 {
		return nil, errors.New("you must instantiate the dag correctly with all fields")
	}
	startNode := dag.Start
	var previousNode DagNode = dag.Start

	previousNode = startNode
	// you can have more than one next node (i.e. for FanOut node)
	currentNodes := previousNode.GetNext()
	previousOutput := input // nil???

	parallel := false // TODO: maybe we need a stack of boolean variables. The top of the stack is the current section

	// while loop to execute dag until we reach the end
	for len(currentNodes) > 0 {
		// execute a single node
		nodeSet := NewNodeSet()
		var nextCurrentNodes []DagNode
		for _, node := range currentNodes {
			// make transition
			errRecv := node.ReceiveInput(previousOutput) // TODO: Retrieve input from ETCD (or from local cache, if the previous node is colocated with the current one)
			if errRecv != nil {
				return nil, fmt.Errorf("the node %s cannot receive the input: %v", node.ToString(), errRecv)
			}
			// handle the output
			// For ChoiceNode, output is sent to first successful branch
			// FIXME: For FanOutNode, output can be a simple copy of previous node output, but here we are calling multiple times
			// FIXME: For FanInNode, output is a merge of all output from back. Todo: How to merge can be a problem... We should also check timeout and exiting if it happens
			var output map[string]interface{}
			if !parallel {
				o, errExec := node.Exec()
				if errExec != nil {
					return nil, fmt.Errorf("the node %s has failed function execution: %v", node.ToString(), errExec)
				}
				output = o
			} else {
				go func() (map[string]interface{}, error) {
					output, err := node.Exec()
					if err != nil {
						return nil, fmt.Errorf("the node %s has failed function execution: %v", node.ToString(), err)
					}
					return output, nil
				}()
			}

			// this wait is necessary to prevent a data race between the storing of a container in the ready pool and the execution of the next node (with a different function)
			switch node.(type) {
			case *SimpleNode:
				<-types.NodeDoneChan
			case *FanOutNode:
				parallel = true
			case *FanInNode:
				parallel = false
			}

			previousOutput = output
			// prepares the output for the next function(s)
			errSend := node.PrepareOutput(output)
			if errRecv != nil {
				return nil, fmt.Errorf("the node %s cannot send the output: %v", node.ToString(), errSend)
			}
			nextNodes := node.GetNext()
			if len(nextNodes) > 0 {
				// adding the next nodes to a set and to a list
				nodeSet.AddAll(nextNodes)
				nextCurrentNodes = append(nextCurrentNodes, nodeSet.GetNodes()...)
			}
		}
		currentNodes = nextCurrentNodes
	}

	return previousOutput, nil
}

// GetUniqueDagFunctions returns a list with the function used in the Dag
func (dag *Dag) GetUniqueDagFunctions() []string {
	allFunctions := make([]string, 0)
	for _, node := range dag.Nodes {
		switch node.(type) {
		case *SimpleNode:
			allFunctions = append(allFunctions, node.(*SimpleNode).Func)
		default:
			continue
		}
	}
	return allFunctions
}

func (dag *Dag) Equals(comparer types.Comparable) bool {

	dag2 := comparer.(*Dag)
	for i := 0; i < len(dag.Nodes); i++ {
		if !dag.Nodes[i].Equals(dag2.Nodes[i]) {
			return false
		}
	}
	return dag.Start == dag2.Start &&
		dag.End == dag2.End &&
		dag.Width == dag2.Width &&
		len(dag.Nodes) == len(dag2.Nodes)
}
