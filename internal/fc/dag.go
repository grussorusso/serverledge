package fc

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
	"math"
	"strings"
)

// used to send output from parallel nodes to fan in node or to the next node
// var outputChannel = make(chan map[string]interface{})

// Dag is a Workflow to drive the execution of the function composition
type Dag struct {
	Start *StartNode // a single start must be added
	Nodes map[DagNodeId]DagNode
	End   *EndNode // a single endNode must be added
	Width int      // width is the max fanOut degree of the Dag
}

func NewDAG() Dag {
	start := NewStartNode()
	end := NewEndNode()
	nodes := make(map[DagNodeId]DagNode)
	nodes[start.Id] = start
	nodes[end.Id] = end

	dag := Dag{
		Start: start,
		End:   end,
		Nodes: nodes,
		Width: 1,
	}
	return dag
}

func (dag *Dag) Find(nodeId DagNodeId) (DagNode, bool) {
	dagNode, found := dag.Nodes[nodeId]
	return dagNode, found
}

// TODO: only the subsequent APIs should be public: NewDag, Print, GetUniqueDagFunctions, Equals
//  the remaining should be private after the builder APIs work well!!!

// addNode can be used to add a new node to the Dag. Does not chain anything, but updates Dag width
func (dag *Dag) addNode(node DagNode) {
	dag.Nodes[node.GetId()] = node // if already exists, overwrites!
	// updates width
	nodeWidth := node.Width()
	if nodeWidth > dag.Width {
		dag.Width = nodeWidth
	}
}

func isDagNodePresent(node DagNode, infos []DagNode) bool {
	isPresent := false
	for _, nodeInfo := range infos {
		if nodeInfo == node {
			isPresent = true
			break
		}
	}
	return isPresent
}

func isEndNode(node DagNode) bool {
	_, ok := node.(*EndNode)
	return ok
}

// VisitDag visits the dag starting from the input node and return a list of visited nodes. If excludeEnd = true, the EndNode will not be in the output list
func VisitDag(dag *Dag, nodeId DagNodeId, nodes []DagNode, excludeEnd bool) []DagNode {
	node, ok := dag.Find(nodeId)
	if !ok {
		return []DagNode{}
	}
	if !isDagNodePresent(node, nodes) {
		nodes = append(nodes, node)
	}
	switch n := node.(type) {
	case *StartNode:
		toAdd := VisitDag(dag, n.GetNext()[0], nodes, excludeEnd)
		for _, add := range toAdd {
			if !isDagNodePresent(add, nodes) {
				// only when isEndNode = true, excludeEnd = true -> we don't add the node
				if !isEndNode(add) || !excludeEnd {
					nodes = append(nodes, add)
				}
			}
		}
		return nodes
	case *SimpleNode:
		toAdd := VisitDag(dag, n.GetNext()[0], nodes, excludeEnd)
		for _, add := range toAdd {
			if !isDagNodePresent(add, nodes) {
				if !isEndNode(add) || !excludeEnd {
					nodes = append(nodes, add)
				}
			}
		}
		return nodes
	case *EndNode:
		if !excludeEnd { // move end node to the end of the visit list
			endNode := n
			// get index of end node to remove
			indexToRemove := -1
			for i, dagNode := range nodes {
				if isEndNode(dagNode) {
					indexToRemove = i
					break
				}
			}
			// remove end node
			nodes = append(nodes[:indexToRemove], nodes[indexToRemove+1:]...)
			// append at the end of the visited node list
			nodes = append(nodes, endNode)
		}
		return nodes
	case *ChoiceNode:
		for _, alternative := range n.Alternatives {
			toAdd := VisitDag(dag, alternative, nodes, excludeEnd)
			for _, add := range toAdd {
				if !isDagNodePresent(add, nodes) {
					if !isEndNode(add) || !excludeEnd {
						nodes = append(nodes, add)
					}
				}
			}
		}
		return nodes
	case *FanOutNode:
		for _, parallelBranch := range n.GetNext() {
			toAdd := VisitDag(dag, parallelBranch, nodes, excludeEnd)
			for _, add := range toAdd {
				if !isDagNodePresent(add, nodes) {
					if !isEndNode(add) || !excludeEnd {
						nodes = append(nodes, add)
					}
				}
			}
		}
		return nodes
	case *FanInNode:
		toAdd := VisitDag(dag, n.GetNext()[0], nodes, excludeEnd)
		for _, add := range toAdd {
			if !isDagNodePresent(add, nodes) {
				if !isEndNode(add) || !excludeEnd {
					nodes = append(nodes, add)
				}
			}
		}
	}
	return nodes
}

// chain can be used to connect the output of node1 to the node2
func (dag *Dag) chain(node1 DagNode, node2 DagNode) error {
	return node1.AddOutput(dag, node2.GetId())
}

// ChainToEndNode (node, i) can be used as a shorthand to chain(node, dag.end[i]) to chain a node to a specific end node
func (dag *Dag) ChainToEndNode(node1 DagNode) error {
	return dag.chain(node1, dag.End)
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
		var currentNodesToAdd []DagNodeId
		for i, nodeId := range currentNodes {
			node, _ := dag.Find(nodeId)
			result += fmt.Sprintf("%s", node.Name())

			doneNodes.AddIfNotExists(node)

			if i != len(currentNodes)-1 {
				result += "|"
			}
			var addNodes []DagNodeId
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

func (dag *Dag) executeStart(progress *Progress, node *StartNode) (bool, error) {
	err := progress.CompleteNode(node.GetId())
	if err != nil {
		return false, err
	}
	return true, nil
}

func (dag *Dag) executeSimple(requestId ReqId, progress *Progress, node *SimpleNode) (bool, error) {
	// retrieving input
	var pd *PartialData
	nodeId := node.GetId()

	input, err := partialDataCache.Retrieve(requestId, nodeId)
	if err != nil {
		return false, err
	}
	err = node.ReceiveInput(input)
	if err != nil {
		return false, err
	}
	// executing node
	output, err := node.Exec(progress)
	if err != nil {
		return false, err
	}
	// this wait is necessary to prevent a data race between the storing of a container in the ready pool and the execution of the next node (with a different function)
	<-types.NodeDoneChan

	pd = NewPartialData(requestId, node.GetNext()[0], nodeId, output)
	errSend := node.PrepareOutput(nil, output)
	if errSend != nil {
		return false, fmt.Errorf("the node %s cannot send the output: %v", node.ToString(), errSend)
	}

	// saving partial data and updating progress
	partialDataCache.Save(pd)
	err = progress.CompleteNode(nodeId)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (dag *Dag) executeChoice(requestId ReqId, progress *Progress, choice *ChoiceNode) (bool, error) {
	// retrieving input
	var pd *PartialData
	nodeId := choice.GetId()

	input, err := partialDataCache.Retrieve(requestId, nodeId)
	if err != nil {
		return false, err
	}
	err = choice.ReceiveInput(input)
	if err != nil {
		return false, err
	}
	// executing node
	output, err := choice.Exec(progress)
	if err != nil {
		return false, err
	}
	pd = NewPartialData(requestId, choice.GetNext()[0], nodeId, output)
	errSend := choice.PrepareOutput(nil, output)
	if errSend != nil {
		return false, fmt.Errorf("the node %s cannot send the output: %v", choice.ToString(), errSend)
	}

	// for choice node, we skip all branch that will not be executed
	nodesToSkip := choice.GetNodesToSkip(dag)
	errSkip := progress.SkipAll(nodesToSkip)
	if errSkip != nil {
		return false, errSkip
	}

	// saving partial data and updating progress
	partialDataCache.Save(pd)
	err = progress.CompleteNode(nodeId)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (dag *Dag) executeParallel(requestId ReqId, progress *Progress, nextNodes []DagNodeId) error {
	// preparing dag nodes and channels for parallel execution
	parallelDagNodes := make([]DagNode, 0)
	outputChannels := make([]chan map[string]interface{}, 0)
	errorChannels := make([]chan error, 0)
	partialDatas := make([]*PartialData, 0)
	for _, nodeId := range nextNodes {
		node, ok := dag.Find(nodeId)
		if ok {
			parallelDagNodes = append(parallelDagNodes, node)
			outputChannels = append(outputChannels, make(chan map[string]interface{}))
			errorChannels = append(errorChannels, make(chan error))
		}
	}
	// executing all nodes in parallel
	for i, node := range parallelDagNodes {
		go func(i int, node DagNode) {
			output, err := node.Exec(progress)
			if err != nil {
				outputChannels[i] <- nil
				errorChannels[i] <- err
				return
			}
			outputChannels[i] <- output
			errorChannels[i] <- nil
		}(i, node)
	}
	// checking errors
	parallelErrors := make([]error, 0)
	for _, errChan := range errorChannels {
		err := <-errChan
		if err != nil {
			parallelErrors = append(parallelErrors, err)
		}
	}
	// retrieving outputs (goroutines should end now)
	parallelOutputs := make([]map[string]interface{}, 0)
	for _, outChan := range outputChannels {
		out := <-outChan
		if out != nil {
			parallelOutputs = append(parallelOutputs, out)
		}
	}
	// returning errors
	if len(parallelErrors) > 0 {
		return fmt.Errorf("errors in parallel execution: %v", parallelErrors)
	}
	// saving partial data
	for i, output := range parallelOutputs {
		node := parallelDagNodes[i]
		pd := NewPartialData(requestId, node.GetNext()[0], node.GetId(), nil)
		partialDatas = append(partialDatas, pd)
		pd.Data = output
		partialDataCache.Save(pd)
		err := progress.CompleteNode(parallelDagNodes[i].GetId())
		if err != nil {
			return err
		}
	}
	return nil
}

// TODO: assicurarsi che si esegua in parallelo
// TODO: aggiungere lo stato di avanzamento: ad esempio su ETCD, compresi i parametri di input/output
func (dag *Dag) Execute(requestId ReqId) (bool, error) {
	/*errChan := make(chan error)
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
				o, errExec := node.Exec() // simple, choice or fanout
				if errExec != nil {
					return nil, fmt.Errorf("the node %s has failed function execution: %v", node.ToString(), errExec)
				}
				output = o
				// this wait is necessary to prevent a data race between the storing of a container in the ready pool and the execution of the next node (with a different function)
				switch node.(type) {
				case *SimpleNode:
					<-types.NodeDoneChan
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
			} else { // if parallel (only if it's not a fan in)
				if _, ok := node.(*FanInNode); !ok {
					// async
					go func() {
						o, err := node.Exec()
						if err != nil {
							errChan <- fmt.Errorf("the node %s has failed function execution: %v", node.ToString(), err)
							return
						}
						// if the next node is fanIn, send the output to it
						errChan <- nil
						if fanIn, ok := node.GetNext()[0].(*FanInNode); ok {
							fmt.Printf("node in branchId %d sent output to fanIn\n", node.GetBranchId())
							fanInChannel := getChannelForParallelBranch(fanIn.Id, node.GetBranchId())
							fanInChannel <- o
						} else {
							// output should go to the next node
							previousOutput = output
							// prepares the output for the next function(s)
							errSend := node.PrepareOutput(output)
							if errRecv != nil {
								errChan <- fmt.Errorf("the node %s cannot send the output: %v", node.ToString(), errSend)
								return
							}
							nextNodes := node.GetNext()
							if len(nextNodes) > 0 {
								// adding the next nodes to a set and to a list
								nodeSet.AddAll(nextNodes)
								nextCurrentNodes = append(nextCurrentNodes, nodeSet.GetNodes()...)
							}
						}
						errChan <- nil
					}()
				}
			}

			switch fan := node.(type) {
			case *FanOutNode:
				parallel = true
				associatedFanIn, _ := dag.Find(fan.AssociatedFanIn)
				nextCurrentNodes = append(nextCurrentNodes, associatedFanIn) // FIXME: problem when there are more than 1 node in each branch
			case *FanInNode:
				parallel = false

				for i := 0; i < fan.FanInDegree; i++ {
					erro := <-errChan
					if erro != nil {
						return nil, erro
					}
				}
				fmt.Println("FanIn: all function returned an output!")
				// fan.PrepareOutput()
				previousOutput = output
				// prepares the output for the next function(s)
				//errSend := node.PrepareOutput(output)
				//if errSend != nil {
				//	return nil, fmt.Errorf("the node %s cannot send the output: %v", node.ToString(), errSend)
				//}
				nextNodes := node.GetNext()
				if len(nextNodes) > 0 {
					// adding the next nodes to a set and to a list
					nodeSet.AddAll(nextNodes)
					nextCurrentNodes = append(nextCurrentNodes, nodeSet.GetNodes()...)
				}
			}
		}
		currentNodes = nextCurrentNodes
	}

	return previousOutput, nil*/

	progress, _ := progressCache.RetrieveProgress(requestId)
	nextNodes, err := progress.NextNodes()
	shouldContinue := true
	// TODO: impostare lo stato del dagNode a Failed in caso di errore. Salvare il messaggio di errore nel progress
	if len(nextNodes) > 1 {
		err := dag.executeParallel(requestId, progress, nextNodes)
		if err != nil {
			return true, err
		}
	} else {
		n, ok := dag.Find(nextNodes[0])
		if !ok {
			return true, fmt.Errorf("failed to find node %s", n.GetId())
		}

		switch node := n.(type) {
		case *SimpleNode:
			shouldContinue, err = dag.executeSimple(requestId, progress, node)
		case *ChoiceNode:
			shouldContinue, err = dag.executeChoice(requestId, progress, node)
		case *FanInNode:
		case *StartNode:
			shouldContinue, err = dag.executeStart(progress, node)
		case *FanOutNode:
		case *EndNode:
			shouldContinue = false
		}
		if err != nil {
			return true, err
		}
	}

	err = progressCache.SaveProgress(progress)
	if err != nil {
		return true, err
	}
	return shouldContinue, nil
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

	for k := range dag.Nodes {
		if dag.Nodes[k] != dag2.Nodes[k] {
			return false
		}
	}
	return dag.Start == dag2.Start &&
		dag.End == dag2.End &&
		dag.Width == dag2.Width &&
		len(dag.Nodes) == len(dag2.Nodes)
}
