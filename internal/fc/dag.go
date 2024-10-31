package fc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/grussorusso/serverledge/internal/asl"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/types"
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
	case *SimpleNode, *PassNode, *WaitNode, *SucceedNode, *FailNode:
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
			// get index of end node to remove\
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

func (dag *Dag) executeStart(progress *Progress, partialData *PartialData, node *StartNode, r *CompositionRequest) (*PartialData, *Progress, bool, error) {

	err := progress.CompleteNode(node.GetId())
	if err != nil {
		return partialData, progress, false, err
	}
	r.ExecReport.Reports.Set(CreateExecutionReportId(node), &function.ExecutionReport{Result: "start"})
	return partialData, progress, true, nil
}

func (dag *Dag) executeSimple(progress *Progress, partialData *PartialData, simpleNode *SimpleNode, r *CompositionRequest) (*PartialData, *Progress, bool, error) {
	// retrieving input
	var pd *PartialData
	nodeId := simpleNode.GetId()
	requestId := ReqId(r.ReqId)
	pd = NewPartialData(requestId, "", nodeId, nil) // partial initialization of pd

	err := simpleNode.CheckInput(partialData.Data)
	if err != nil {
		return pd, progress, false, err
	}
	// executing node
	output, err := simpleNode.Exec(r, partialData.Data)
	if err != nil {
		return pd, progress, false, err
	}

	// Todo: uncomment when running TestInvokeFC_Concurrent to debug concurrency errors
	// errDbg := Debug(r, string(simpleNode.Id), output)
	// if errDbg != nil {
	// 	return false, errDbg
	// }

	forNode := simpleNode.GetNext()[0]

	errSend := simpleNode.PrepareOutput(dag, output)
	if errSend != nil {
		return pd, progress, false, fmt.Errorf("the node %s cannot send the output: %v", simpleNode.String(), errSend)
	}

	// setting the remaining fields of pd
	pd.ForNode = forNode
	pd.Data = output

	err = progress.CompleteNode(nodeId)
	if err != nil {
		return pd, progress, false, err
	}

	return pd, progress, true, nil
}

func (dag *Dag) executeChoice(progress *Progress, partialData *PartialData, choice *ChoiceNode, r *CompositionRequest) (*PartialData, *Progress, bool, error) {

	var pd *PartialData
	nodeId := choice.GetId()
	requestId := ReqId(r.ReqId)
	pd = NewPartialData(requestId, "", nodeId, nil) // partial initialization of pd

	err := choice.CheckInput(partialData.Data)
	if err != nil {
		return pd, progress, false, err
	}
	// executing node
	output, err := choice.Exec(r, partialData.Data)
	if err != nil {
		return pd, progress, false, err
	}

	// setting the remaining fields of pd
	pd.ForNode = choice.GetNext()[0]
	pd.Data = output

	errSend := choice.PrepareOutput(dag, output)
	if errSend != nil {
		return pd, progress, false, fmt.Errorf("the node %s cannot send the output: %v", choice.String(), errSend)
	}

	// for choice node, we skip all branch that will not be executed
	nodesToSkip := choice.GetNodesToSkip(dag)
	errSkip := progress.SkipAll(nodesToSkip)
	if errSkip != nil {
		return pd, progress, false, errSkip
	}

	err = progress.CompleteNode(nodeId)
	if err != nil {
		return pd, progress, false, err
	}

	return pd, progress, true, nil
}

func (dag *Dag) executeFanOut(progress *Progress, partialData *PartialData, fanOut *FanOutNode, r *CompositionRequest) (*PartialData, *Progress, bool, error) {

	var pd *PartialData
	outputMap := make(map[string]interface{})
	nodeId := fanOut.GetId()
	requestId := ReqId(r.ReqId)

	/* using forNode = "" in order to create a special partialData to handle fanout
	 * case with Data field which contains a map[string]interface{} with the key set
	 * to nodeId and the value which is also a map[string]interface{} containing the
	 * effective input for the nth-parallel node */
	pd = NewPartialData(requestId, "", nodeId, nil) // partial initialization of pd

	// executing node
	output, err := fanOut.Exec(r, partialData.Data)
	if err != nil {
		return pd, progress, false, err
	}
	// sends output to each next node
	errSend := fanOut.PrepareOutput(dag, output)
	if errSend != nil {
		return pd, progress, false, fmt.Errorf("the node %s cannot send the output: %v", fanOut.String(), errSend)
	}

	for i, nextNode := range fanOut.GetNext() {
		if fanOut.Type == Broadcast {
			outputMap[fmt.Sprintf("%s", nextNode)] = output[fmt.Sprintf("%d", i)].(map[string]interface{})
		} else if fanOut.Type == Scatter {
			firstName := ""
			for name := range output {
				firstName = name
				break
			}
			inputForNode := make(map[string]interface{})
			subMap, found := output[firstName].(map[string]interface{})
			if !found {
				return pd, progress, false, fmt.Errorf("cannot find parameter for nextNode %s", nextNode)
			}
			inputForNode[firstName] = subMap[fmt.Sprintf("%d", i)]
			outputMap[fmt.Sprintf("%s", nextNode)] = inputForNode
		} else {
			return pd, progress, false, fmt.Errorf("invalid fanout type %d", fanOut.Type)
		}
	}

	// setting the remaining field of pd
	pd.Data = outputMap
	// and updating progress
	err = progress.CompleteNode(nodeId)
	if err != nil {
		return pd, progress, false, err
	}
	return pd, progress, true, nil
}

func (dag *Dag) executeParallel(progress *Progress, partialData *PartialData, nextNodes []DagNodeId, r *CompositionRequest) (*PartialData, *Progress, error) {
	// preparing dag nodes and channels for parallel execution
	parallelDagNodes := make([]DagNode, 0)
	inputs := make([]map[string]interface{}, 0)
	outputChannels := make([]chan map[string]interface{}, 0)
	errorChannels := make([]chan error, 0)
	requestId := ReqId(r.ReqId)
	outputMap := make(map[string]interface{}, 0)
	var node DagNode
	pd := NewPartialData(requestId, "", "", nil) // partial initialization of pd

	for _, nodeId := range nextNodes {
		node, ok := dag.Find(nodeId)
		if ok {
			parallelDagNodes = append(parallelDagNodes, node)
			outputChannels = append(outputChannels, make(chan map[string]interface{}))
			errorChannels = append(errorChannels, make(chan error))
		}
		// for simple node we also retrieve the partial data and receive input
		if simple, isSimple := node.(*SimpleNode); isSimple {
			errInput := simple.CheckInput(partialData.Data[fmt.Sprintf("%s", nodeId)].(map[string]interface{}))
			if errInput != nil {
				return pd, progress, errInput
			}
			inputs = append(inputs, partialData.Data[fmt.Sprintf("%s", nodeId)].(map[string]interface{}))
		}
	}
	// executing all nodes in parallel
	for i, node := range parallelDagNodes {
		go func(i int, params map[string]interface{}, node DagNode) {
			output, err := node.Exec(r, params)
			// for simple node, we also prepare output
			if simpleNode, isSimple := node.(*SimpleNode); isSimple {
				errSend := simpleNode.PrepareOutput(dag, output)
				if errSend != nil {
					errorChannels[i] <- err
					outputChannels[i] <- nil
					return
				}
			}
			// first send on error, then on output channels
			if err != nil {
				errorChannels[i] <- err
				outputChannels[i] <- nil
				return
			}
			errorChannels[i] <- nil
			outputChannels[i] <- output
			// fmt.Printf("goroutine %d for node %s completed\n", i, node.GetId())
		}(i, inputs[i], node)
	}
	// checking errors
	parallelErrors := make([]error, 0)
	for _, errChan := range errorChannels {
		err := <-errChan
		if err != nil {
			parallelErrors = append(parallelErrors, err)
			// we do not return now, because we want to quit the goroutines
			// we also need to check the outputs.
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
		return pd, progress, fmt.Errorf("errors in parallel execution: %v", parallelErrors)
	}

	for i, output := range parallelOutputs {
		node = parallelDagNodes[i]
		outputMap[fmt.Sprintf("%s", node.GetId())] = output
		err := progress.CompleteNode(parallelDagNodes[i].GetId())
		if err != nil {
			return pd, progress, err
		}
	}
	/* using fromNode = "" in order to create a special partialData to handle parallel case with
	 * Data field which contains a map[string]interface{} with the key set to nodeId
	 * and the value which is also a map[string]interface{} containing the effective
	 * output of the nth-parallel node */
	//pd := NewPartialData(requestId, node.GetNext()[0], "", outputMap)
	// setting the remaining fields of pd
	pd.ForNode = node.GetNext()[0]
	pd.Data = outputMap
	return pd, progress, nil
}

func (dag *Dag) executeFanIn(progress *Progress, partialData *PartialData, fanIn *FanInNode, r *CompositionRequest) (*PartialData, *Progress, bool, error) {
	nodeId := fanIn.GetId()
	requestId := ReqId(r.ReqId)
	var err error
	pd := NewPartialData(requestId, "", nodeId, nil) // partial initialization of pd

	// TODO: are you sure it is necessary?
	//err := progress.PutInWait(fanIn)
	//if err != nil {
	//	return false, err
	//}

	timerElapsed := false
	timer := time.AfterFunc(fanIn.Timeout, func() {
		fmt.Println("timeout elapsed")
		timerElapsed = true
	})

	for !timerElapsed {
		if len(partialData.Data) == fanIn.FanInDegree {
			break
		}
		fmt.Printf("fanin waiting partial datas: %d/%d\n", len(partialData.Data), fanIn.FanInDegree)
		time.Sleep(fanIn.Timeout / 100)
	}

	fired := timer.Stop()
	if !fired {
		return pd, progress, false, fmt.Errorf("fan in timeout occurred")
	}
	faninInputs := make([]map[string]interface{}, 0)
	for _, partialDataMap := range partialData.Data {
		faninInputs = append(faninInputs, partialDataMap.(map[string]interface{}))
	}

	// merging input into one output
	output, err := fanIn.Exec(r, faninInputs...)
	if err != nil {
		return pd, progress, false, err
	}

	// setting the remaining field of pd
	pd.ForNode = fanIn.GetNext()[0]
	pd.Data = output
	err = progress.CompleteNode(nodeId)
	if err != nil {
		return pd, progress, false, err
	}

	return pd, progress, true, nil
}

func (dag *Dag) executeSucceedNode(progress *Progress, partialData *PartialData, succeedNode *SucceedNode, r *CompositionRequest) (*PartialData, *Progress, bool, error) {
	return commonExec(dag, progress, partialData, succeedNode, r)
}

func (dag *Dag) executeFailNode(progress *Progress, partialData *PartialData, failNode *FailNode, r *CompositionRequest) (*PartialData, *Progress, bool, error) {
	return commonExec(dag, progress, partialData, failNode, r)
}

func commonExec(dag *Dag, progress *Progress, partialData *PartialData, node DagNode, r *CompositionRequest) (*PartialData, *Progress, bool, error) {
	var pd *PartialData
	nodeId := node.GetId()
	requestId := ReqId(r.ReqId)
	pd = NewPartialData(requestId, "", nodeId, nil) // partial initialization of pd

	err := node.CheckInput(partialData.Data)
	if err != nil {
		return pd, progress, false, err
	}
	// executing node
	output, err := node.Exec(r, partialData.Data)
	if err != nil {
		return pd, progress, false, err
	}

	// Todo: uncomment when running TestInvokeFC_Concurrent to debug concurrency errors
	// errDbg := Debug(r, string(node.Id), output)
	// if errDbg != nil {
	// 	return false, errDbg
	// }

	errSend := node.PrepareOutput(dag, output)
	if errSend != nil {
		return pd, progress, false, fmt.Errorf("the node %s cannot send the output: %v", node.String(), errSend)
	}

	forNode := node.GetNext()[0]
	// setting the remaining fields of pd
	pd.ForNode = forNode
	pd.Data = output

	err = progress.CompleteNode(nodeId)
	if err != nil {
		return pd, progress, false, err
	}
	if node.GetNodeType() == Fail || node.GetNodeType() == Succeed {
		return pd, progress, false, nil
	}
	return pd, progress, true, nil
}

func (dag *Dag) executeEnd(progress *Progress, partialData *PartialData, node *EndNode, r *CompositionRequest) (*PartialData, *Progress, bool, error) {
	r.ExecReport.Reports.Set(CreateExecutionReportId(node), &function.ExecutionReport{Result: "end"})
	err := progress.CompleteNode(node.Id)
	if err != nil {
		return partialData, progress, false, err
	}
	return partialData, progress, false, nil // false because we want to stop when reaching the end
}

func (dag *Dag) Execute(r *CompositionRequest, data *PartialData, progress *Progress) (*PartialData, *Progress, bool, error) {

	var pd *PartialData
	nextNodes, err := progress.NextNodes()
	if err != nil {
		return data, progress, false, fmt.Errorf("failed to get next nodes from progress: %v", err)
	}
	shouldContinue := true

	if len(nextNodes) > 1 {
		pd, progress, err = dag.executeParallel(progress, data, nextNodes, r)
		if err != nil {
			return pd, progress, true, err
		}
	} else if len(nextNodes) == 1 {
		n, ok := dag.Find(nextNodes[0])
		if !ok {
			return data, progress, true, fmt.Errorf("failed to find node %s", n.GetId())
		}

		switch node := n.(type) {
		case *SimpleNode:
			pd, progress, shouldContinue, err = dag.executeSimple(progress, data, node, r)
		case *ChoiceNode:
			pd, progress, shouldContinue, err = dag.executeChoice(progress, data, node, r)
		case *FanInNode:
			pd, progress, shouldContinue, err = dag.executeFanIn(progress, data, node, r)
		case *StartNode:
			pd, progress, shouldContinue, err = dag.executeStart(progress, data, node, r)
		case *FanOutNode:
			pd, progress, shouldContinue, err = dag.executeFanOut(progress, data, node, r)
		case *PassNode:
			pd, progress, shouldContinue, err = commonExec(dag, progress, data, node, r)
		case *WaitNode:
			pd, progress, shouldContinue, err = commonExec(dag, progress, data, node, r)
		case *FailNode:
			pd, progress, shouldContinue, err = dag.executeFailNode(progress, data, node, r) // TODO: use commonExec
		case *SucceedNode:
			pd, progress, shouldContinue, err = dag.executeSucceedNode(progress, data, node, r) // TODO: use commonExec
		case *EndNode:
			pd, progress, shouldContinue, err = dag.executeEnd(progress, data, node, r)
		}
		if err != nil {
			_ = progress.FailNode(n.GetId())
			r.ExecReport.Progress = progress
			return pd, progress, true, err
		}
	} else {
		return data, progress, false, fmt.Errorf("there aren't next nodes")
	}

	return pd, progress, shouldContinue, nil
}

// GetUniqueDagFunctions returns a list with the function names used in the Dag. The returned function names are unique and in alphabetical order
func (dag *Dag) GetUniqueDagFunctions() []string {
	allFunctionsMap := make(map[string]void)
	for _, node := range dag.Nodes {
		switch n := node.(type) {
		case *SimpleNode:
			allFunctionsMap[n.Func] = null
		default:
			continue
		}
	}
	uniqueFunctions := make([]string, 0, len(allFunctionsMap))
	for fName := range allFunctionsMap {
		uniqueFunctions = append(uniqueFunctions, fName)
	}
	// we sort the list to always get the same result
	sort.Strings(uniqueFunctions)

	return uniqueFunctions
}

func (dag *Dag) Equals(comparer types.Comparable) bool {

	dag2 := comparer.(*Dag)

	for k := range dag.Nodes {
		if !dag.Nodes[k].Equals(dag2.Nodes[k]) {
			return false
		}
	}
	return dag.Start.Equals(dag2.Start) &&
		dag.End.Equals(dag2.End) &&
		dag.Width == dag2.Width &&
		len(dag.Nodes) == len(dag2.Nodes)
}

func (dag *Dag) String() string {
	return fmt.Sprintf(`Dag{
		Start: %s,
		Nodes: %s,
		End:   %s,
		Width: %d,
	}`, dag.Start.String(), dag.Nodes, dag.End.String(), dag.Width)
}

// MarshalJSON is needed because DagNode is an interface
func (dag *Dag) MarshalJSON() ([]byte, error) {
	// Create a map to hold the JSON representation of the Dag
	data := make(map[string]interface{})

	// Add the field to the map
	data["Start"] = dag.Start
	data["End"] = dag.End
	data["Width"] = dag.Width
	nodes := make(map[DagNodeId]interface{})

	// Marshal the interface and store it as concrete node value in the map
	for nodeId, node := range dag.Nodes {
		nodes[nodeId] = node
	}
	data["Nodes"] = nodes

	// Marshal the map to JSON
	return json.Marshal(data)
}

// UnmarshalJSON is needed because DagNode is an interface
func (dag *Dag) UnmarshalJSON(data []byte) error {
	// Create a temporary map to decode the JSON data
	var tempMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &tempMap); err != nil {
		return err
	}
	// extract simple fields
	if rawStart, ok := tempMap["Start"]; ok {
		if err := json.Unmarshal(rawStart, &dag.Start); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("missing 'Start' field in JSON")
	}

	if rawEnd, ok := tempMap["End"]; ok {
		if err := json.Unmarshal(rawEnd, &dag.End); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("missing 'End' field in JSON")
	}

	if rawWidth, ok := tempMap["Width"]; ok {
		if err := json.Unmarshal(rawWidth, &dag.Width); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("missing 'Width' field in JSON")
	}

	// Cycle on each map entry and decode the type
	var tempNodeMap map[string]json.RawMessage
	if err := json.Unmarshal(tempMap["Nodes"], &tempNodeMap); err != nil {
		return err
	}
	dag.Nodes = make(map[DagNodeId]DagNode)
	for nodeId, value := range tempNodeMap {
		err := dag.decodeNode(nodeId, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (dag *Dag) decodeNode(nodeId string, value json.RawMessage) error {
	var tempNodeMap map[string]interface{}
	if err := json.Unmarshal(value, &tempNodeMap); err != nil {
		return err
	}
	dagNodeType, ok := tempNodeMap["NodeType"].(string)
	if !ok {
		return fmt.Errorf("unknown nodeType: %v", tempNodeMap["NodeType"])
	}
	var err error

	node := DagNodeFromType(DagNodeType(dagNodeType))

	switch DagNodeType(dagNodeType) {
	case Start:
		node := &StartNode{}
		err = json.Unmarshal(value, node)
		if err == nil && node.Id != "" && node.Next != "" {
			dag.Nodes[DagNodeId(nodeId)] = node
			return nil
		}
	case Simple:
		node := &SimpleNode{}
		err = json.Unmarshal(value, node)
		if err == nil && node.Id != "" && node.Func != "" {
			dag.Nodes[DagNodeId(nodeId)] = node
			return nil
		}
	case Choice:
		node := &ChoiceNode{}
		err = json.Unmarshal(value, node)
		if err == nil && node.Id != "" && node.Alternatives != nil && len(node.Alternatives) == len(node.Conditions) {
			dag.Nodes[DagNodeId(nodeId)] = node
			return nil
		}
	default:
		err = json.Unmarshal(value, node)
		if err == nil && node.GetId() != "" {
			dag.Nodes[DagNodeId(nodeId)] = node
			return nil
		}
	}
	var unmarshalTypeError *json.UnmarshalTypeError
	if err != nil && !errors.As(err, &unmarshalTypeError) {
		// abort if we have an error other than the wrong type
		return err
	}

	return fmt.Errorf("failed to decode node")
}

// IsEmpty returns true if the dag has 0 nodes or exactly one StartNode and one EndNode.
func (dag *Dag) IsEmpty() bool {
	if len(dag.Nodes) == 0 {
		return true
	}

	onlyTwoNodes := len(dag.Nodes) == 2
	hasOnlyStartAndEnd := false
	if onlyTwoNodes {
		hasStart := 0
		hasEnd := 0
		for _, node := range dag.Nodes {
			if node.GetNodeType() == Start {
				hasStart++
			}
			if node.GetNodeType() == End {
				hasEnd++
			}
		}
		hasOnlyStartAndEnd = (hasStart == 1) && (hasEnd == 1)
	}

	if hasOnlyStartAndEnd {
		return true
	}

	return false
}

// Debug can be used to find if expected output of test TestInvokeFC_Concurrent is correct based on requestId of format "goroutine_#" and simple node ids of format "simple #"
func Debug(r *CompositionRequest, nodeId string, output map[string]interface{}) error {
	if strings.Contains(r.ReqId, "goroutine") {
		// getting number of goroutine, to get the starting input of the sequence
		startingInput := strings.Split(r.ReqId, "_")[1]
		atoi, errAtoi1 := strconv.Atoi(startingInput)
		if errAtoi1 != nil {
			return errAtoi1
		}
		// getting the number of simple node, to get the increment value up to this node in the sequence
		currentIncrementMinusOne := strings.Split(nodeId, " ")[1]
		atoiInc, errAtoi2 := strconv.Atoi(currentIncrementMinusOne)
		if errAtoi2 != nil {
			return errAtoi2
		}
		expectedOutput := atoi + atoiInc + 1

		// getting the key of the single map entry
		key := ""
		for name := range output {
			key = name
			break
		}
		// assertEquals
		actualOutput := output[key]
		if expectedOutput != actualOutput {
			contents := GetCacheContents()
			fmt.Println(contents)
			return fmt.Errorf("expected: %d - actual: %v\n", expectedOutput, output)
		}
	}
	return nil
}

func DagBuildingLoop(sm *asl.StateMachine, nextState asl.State, nextStateName string) (*Dag, error) {
	builder := NewDagBuilder()
	isTerminal := false
	// forse questo va messo in un metodo a parte e riutilizzato per navigare i branch dei choice
	for !isTerminal {

		switch nextState.GetType() {
		case asl.Task:

			taskState := nextState.(*asl.TaskState)
			b, err := BuildFromTaskState(builder, taskState, nextStateName)
			if err != nil {
				return nil, fmt.Errorf("failed building SimpleNode from task state: %v", err)
			}
			builder = b
			nextState, nextStateName, isTerminal = findNextOrTerminate(taskState, sm)
			break
		case asl.Parallel:
			parallelState := nextState.(*asl.ParallelState)
			b, err := BuildFromParallelState(builder, parallelState, nextStateName)
			if err != nil {
				return nil, fmt.Errorf("failed building FanInNode and FanOutNode from ParallelState: %v", err)
			}
			builder = b
			nextState, nextStateName, isTerminal = findNextOrTerminate(parallelState, sm)
			break
		case asl.Map:
			mapState := nextState.(*asl.MapState)
			b, err := BuildFromMapState(builder, mapState, nextStateName)
			if err != nil {
				return nil, fmt.Errorf("failed building MapNode from Map state: %v", err) // TODO: MapNode doesn't exist
			}
			builder = b
			nextState, nextStateName, isTerminal = findNextOrTerminate(mapState, sm)
			break
		case asl.Pass:
			passState := nextState.(*asl.PassState)
			b, err := BuildFromPassState(builder, passState, nextStateName)
			if err != nil {
				return nil, fmt.Errorf("failed building SimplNode with function 'pass' from Pass state: %v", err)
			}
			builder = b
			nextState, nextStateName, isTerminal = findNextOrTerminate(passState, sm)
			break
		case asl.Wait:
			waitState := nextState.(*asl.WaitState)
			b, err := BuildFromWaitState(builder, waitState, nextStateName)
			if err != nil {
				return nil, fmt.Errorf("failed building SimpleNode with function 'wait' from Wait state: %v", err)
			}
			builder = b
			nextState, nextStateName, isTerminal = findNextOrTerminate(waitState, sm)
			break
		case asl.Choice:
			choiceState := nextState.(*asl.ChoiceState)
			// In this case, the choice state will automatically build the dag, because it is terminal
			return BuildFromChoiceState(builder, choiceState, nextStateName, sm)
		case asl.Succeed:
			succeed := nextState.(*asl.SucceedState)
			return BuildFromSucceedState(builder, succeed, nextStateName)
		case asl.Fail:
			failState := nextState.(*asl.FailState)
			return BuildFromFailState(builder, failState, nextStateName)
		default:
			return nil, fmt.Errorf("unknown state type %s", nextState.GetType())
		}
	}
	return builder.Build()
}

func FromStateMachine(sm *asl.StateMachine) (*Dag, error) {
	nextStateName := sm.StartAt
	nextState := sm.States[nextStateName]
	return DagBuildingLoop(sm, nextState, nextStateName)
}

// findNextOrTerminate returns the State, its name and if it is terminal or not
func findNextOrTerminate(state asl.CanEnd, sm *asl.StateMachine) (asl.State, string, bool) {
	isTerminal := state.IsEndState()
	var nextState asl.State = nil
	var nextStateName string = ""

	if !isTerminal {
		nextName, ok := state.(asl.HasNext).GetNext()
		if !ok {
			return nil, "", true
		}
		nextStateName = nextName
		nextState = sm.States[nextStateName]
	}
	return nextState, nextStateName, isTerminal
}
