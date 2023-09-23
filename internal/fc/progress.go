package fc

import (
	"fmt"
	"math"
)

type ReqId string
type ReqId_DagNodeId string

var progressCache = make(map[ReqId]*Progress)
var partialDataCache = make(map[ReqId_DagNodeId]*PartialData)

// TODO: add progress to FunctionComposition Request (maybe doesn't exists)
// Progress tracks the progress of a Dag, i.e. which nodes are executed, and what is the next node to run. Dag progress is saved in ETCD and retrieved by the next node
type Progress struct {
	ReqId    string // requestId, used to distinguish different dag's progresses
	DagNodes []*NodeInfo
	// NextNodeId string // id of next dagNode to execute // FIXME: Maybe useless
}

type NodeInfo struct {
	Id     string
	Type   DagNodeType
	Status DagNodeStatus
	Group  int // The group helps represent the order of execution of nodes. Nodes with the same group should run concurrently
	Branch int // copied from dagNode
}

func newNodeInfo(dNode DagNode, group int) *NodeInfo {
	return &NodeInfo{
		Id:     dNode.GetId(),
		Type:   parseType(dNode),
		Status: Pending,
		Group:  group,
		Branch: dNode.GetBranchId(),
	}
}

type DagNodeStatus int

const (
	Pending = iota
	Executed
	Skipped // TODO: if a node is skipped, when invoking the next node, we automatically skip it without executing, until we reach the end (that should not be skipped)
	Failed
)

func printStatus(s DagNodeStatus) string {
	switch s {
	case Pending:
		return "Pending"
	case Executed:
		return "Executed"
	case Skipped:
		return "Skipped"
	case Failed:
		return "Failed"
	}
	return "No Status - Error"
}

type DagNodeType int

const (
	Start = iota
	End
	Simple
	Choice
	FanOut
	FanIn
)

func parseType(dNode DagNode) DagNodeType {
	switch dNode.(type) {
	case *StartNode:
		return Start
	case *EndNode:
		return End
	case *SimpleNode:
		return Simple
	case *ChoiceNode:
		return Choice
	case *FanOutNode:
		return FanOut
	case *FanInNode:
		return FanIn
	}
	panic("unreachable!")
}
func printType(t DagNodeType) string {
	switch t {
	case Start:
		return "Start"
	case End:
		return "End"
	case Simple:
		return "Simple"
	case Choice:
		return "Choice"
	case FanOut:
		return "FanOut"
	case FanIn:
		return "FanIn"
	}
	return ""
}

func (p *Progress) IsCompleted() bool {
	for _, node := range p.DagNodes {
		if node.Status == Pending {
			return false
		}
	}
	return true

}

// NextNodes retrieves the next nodes to execute, that have the minimum group with state pending
func (p *Progress) NextNodes() []string {
	minPendingGroup := -1
	// find the min group with node pending
	for _, node := range p.DagNodes {
		if node.Status == Pending {
			minPendingGroup = node.Group
			break
		}
	}
	// get all node Ids within that group
	nodeIds := make([]string, 0)
	for _, node := range p.DagNodes {
		if node.Group == minPendingGroup && node.Status == Pending {
			nodeIds = append(nodeIds, node.Id)
		}
	}
	return nodeIds
}

func (p *Progress) CompleteNode(id string) error {
	for _, node := range p.DagNodes {
		if node.Id == id {
			node.Status = Executed
			return nil
		}
	}
	return fmt.Errorf("no node to complete with id %s exists in the dag for request %s", id, p.ReqId)
}

func (p *Progress) SkipNode(id string) error {
	for _, node := range p.DagNodes {
		if node.Id == id {
			node.Status = Skipped
			fmt.Printf("skipped node %s\n", id)
			return nil
		}
	}
	return fmt.Errorf("no node to skip with id %s exists in the dag for request %s", id, p.ReqId)
}

func (p *Progress) SkipAll(nodes []DagNode) error {
	for _, node := range nodes {
		err := p.SkipNode(node.GetId())
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Progress) FailNode(id string) error {
	for _, node := range p.DagNodes {
		if node.Id == id {
			node.Status = Failed
			return nil
		}
	}
	return fmt.Errorf("no node to fail with id %s exists in the dag for request %s", id, p.ReqId)
}

func (p *Progress) GetInfo(nodeId string) *NodeInfo {
	for _, node := range p.DagNodes {
		if node.Id == nodeId {
			return node
		}
	}
	return nil
}

func (p *Progress) GetGroup(nodeId string) int {
	for _, node := range p.DagNodes {
		if node.Id == nodeId {
			return node.Group
		}
	}
	return -1
}

// PartialData is saved separately from progressData to avoid cluttering the Progress struct and each Serverledge node's cache
type PartialData struct {
	ReqId    string // request referring to this partial data
	ForNode  string // dagNode that should receive this partial data
	FromNode string // TODO: maybe useless
	Data     map[string]interface{}
}

// moveEndNodeAtTheEnd moves the end node at the end of the list and sets its group accordingly
func moveEndNodeAtTheEnd(nodeInfos []*NodeInfo) []*NodeInfo {
	// move the endNode at the end of the list
	var endNodeInfo *NodeInfo
	// get index of end node to remove
	indexToRemove := -1
	maxGroup := 0
	for i, nodeInfo := range nodeInfos {
		if nodeInfo.Type == End {
			indexToRemove = i
			endNodeInfo = nodeInfo
			continue
		}
		if nodeInfo.Group > maxGroup {
			maxGroup = nodeInfo.Group
		}
	}
	if indexToRemove != -1 {
		// remove end node
		nodeInfos = append(nodeInfos[:indexToRemove], nodeInfos[indexToRemove+1:]...)
		// update endNode group
		endNodeInfo.Group = maxGroup + 1
		// append at the end of the visited node list
		nodeInfos = append(nodeInfos, endNodeInfo)
	}
	return nodeInfos
}

// InitProgressRecursive initialize the node list assigning a group to each node, so that we can know which nodes should run in parallel or is a choice branch
func InitProgressRecursive(reqId string, dag *Dag) *Progress {
	nodeInfos := extractNodeInfo(dag.Start, 0, make([]*NodeInfo, 0))
	nodeInfos = moveEndNodeAtTheEnd(nodeInfos)
	nodeInfos = reorder(nodeInfos)
	return &Progress{
		ReqId:    reqId,
		DagNodes: nodeInfos,
	}
}

// popMinGroupAndBranchNode removes the node with minimum group and, in case of multiple nodes in the same group, minimum branch
func popMinGroupAndBranchNode(infos *[]*NodeInfo) *NodeInfo {
	// finding min group nodes
	minGroup := math.MaxInt
	var minGroupNodeInfo []*NodeInfo
	for _, info := range *infos {
		if info.Group < minGroup {
			minGroupNodeInfo = make([]*NodeInfo, 0)
			minGroup = info.Group
			minGroupNodeInfo = append(minGroupNodeInfo, info)
		}
		if info.Group == minGroup {
			minGroupNodeInfo = append(minGroupNodeInfo, info)
		}
	}
	minBranch := math.MaxInt // when there are ties
	var minGroupAndBranchNode *NodeInfo

	// finding min branch node from those of the minimum group
	for _, info := range minGroupNodeInfo {
		if info.Branch < minBranch {
			minBranch = info.Branch
			minGroupAndBranchNode = info
		}
	}

	// finding index to remove from starting list
	var indexToRemove int
	for i, info := range *infos {
		if info.Id == minGroupAndBranchNode.Id {
			indexToRemove = i
			break
		}
	}
	*infos = append((*infos)[:indexToRemove], (*infos)[indexToRemove+1:]...)
	return minGroupAndBranchNode
}

func reorder(infos []*NodeInfo) []*NodeInfo {
	reordered := make([]*NodeInfo, 0)
	fmt.Println(len(reordered))
	for len(infos) > 0 {
		next := popMinGroupAndBranchNode(&infos)
		reordered = append(reordered, next)
	}
	return reordered
}

func isNodeInfoPresent(node string, infos []*NodeInfo) bool {
	isPresent := false
	for _, nodeInfo := range infos {
		if nodeInfo.Id == node {
			isPresent = true
			break
		}
	}
	return isPresent
}

// extractNodeInfo retrieves all needed information from nodes and sets node groups. It duplicates end nodes.
func extractNodeInfo(node DagNode, group int, infos []*NodeInfo) []*NodeInfo {
	info := newNodeInfo(node, group)
	if !isNodeInfoPresent(node.GetId(), infos) {
		infos = append(infos, info)
	} else if n, ok := node.(*FanInNode); ok {
		for _, nodeInfo := range infos {
			if nodeInfo.Id == n.GetId() {
				nodeInfo.Group = group
				break
			}
		}
	}
	group++
	switch n := node.(type) {
	case *StartNode:
		toAdd := extractNodeInfo(n.GetNext()[0], group, infos)
		for _, add := range toAdd {
			if !isNodeInfoPresent(add.Id, infos) {
				infos = append(infos, add)
			}
		}
		return infos
	case *SimpleNode:
		toAdd := extractNodeInfo(n.GetNext()[0], group, infos)
		for _, add := range toAdd {
			if !isNodeInfoPresent(add.Id, infos) {
				infos = append(infos, add)
			}
		}
		return infos
	case *EndNode:
		return infos
	case *ChoiceNode:
		for _, alternative := range n.Alternatives {
			toAdd := extractNodeInfo(alternative, group, infos)
			for _, add := range toAdd {
				if !isNodeInfoPresent(add.Id, infos) {
					infos = append(infos, add)
				}
			}
		}
		return infos
	case *FanOutNode:
		for _, parallelBranch := range n.GetNext() {
			toAdd := extractNodeInfo(parallelBranch, group, infos)
			for _, add := range toAdd {
				if !isNodeInfoPresent(add.Id, infos) {
					infos = append(infos, add)
				}
			}
		}
		return infos
	case *FanInNode:
		toAdd := extractNodeInfo(n.GetNext()[0], group, infos)
		for _, add := range toAdd {
			if !isNodeInfoPresent(add.Id, infos) {
				infos = append(infos, add)
			}
		}
	}
	return infos
}

// InitProgress initialize the node list assigning a group to each node, so that we can know which nodes should run in parallel
//func InitProgress(reqId string, dag *Dag) *Progress {
//	group := 0
//	// the list to return
//	nodeInfos := make([]*NodeInfo, 0)
//	// auxiliary list to assign the same group each to node it contains
//	currentNodeGroup := make([]DagNode, 0)
//	currentNodeGroup = append(currentNodeGroup, dag.Start)
//	info := newNodeInfo(dag.Start, group)
//	nodeInfos = append(nodeInfos, info)
//	// while there are nodes
//	for i := 0; i < len(dag.Nodes)+2; i++ {
//		group++
//		// lets get the first node in the group, for example StartNode
//		node := currentNodeGroup[0]
//		// we delete the node, because its group is already set
//		currentNodeGroup = slices.Delete(currentNodeGroup, 0, 1)
//		// we retrieve the next nodes
//		var nextNodes []DagNode
//		switch n := node.(type) {
//		case *ChoiceNode:
//			nextNodes = n.Alternatives
//		case *EndNode:
//			continue // only if
//		default:
//			nextNodes = n.GetNext()
//		}
//		// and add them to the group list, because they are all in the same group
//		currentNodeGroup = append(currentNodeGroup, nextNodes...)
//
//		// we create the NodeInfo structs and set the same group to each of them
//		for _, groupNode := range currentNodeGroup {
//			info := newNodeInfo(groupNode, group)
//			alreadyPresent := false
//			for _, nodeInfo := range nodeInfos {
//				if info.Id == nodeInfo.Id {
//					alreadyPresent = true
//					break
//				}
//			}
//			if !alreadyPresent {
//				nodeInfos = append(nodeInfos, info)
//			}
//		}
//		// reset the auxiliary list
//		currentNodeGroup = make([]DagNode, 0)
//		currentNodeGroup = append(currentNodeGroup, nextNodes...)
//	}
//
//	return &Progress{
//		// ReqId:    reqId,
//		DagNodes: nodeInfos,
//	}
//}

func (p *Progress) Print() {
	str := fmt.Sprintf("Progress for composition request %s - G = node group, B = node branch\n", p.ReqId)
	str += fmt.Sprintln("G. |B| Type   (        NodeID        ) - Status")
	str += fmt.Sprintln("-------------------------------------------------")
	for _, info := range p.DagNodes {
		str += fmt.Sprintf("%d. |%d| %-6s (%-22s) - %s\n", info.Group, info.Branch, printType(info.Type), info.Id, printStatus(info.Status))
	}
	fmt.Printf("%s", str)
}

// Update should be used by a completed node after its execution
//func Update(p *Progress, s DagNodeStatus, n string, next n) {
//	p.doneNodes++ // TODO: how to deal with choice nodes?
//}

// SaveProgress should be used by a completed node after its execution
func SaveProgress(p *Progress) error {
	// TODO: save progress in ETCD
	return nil

}

// RetrieveProgress should be used by the next node to execute
func RetrieveProgress(reqId string) *Progress {
	// TODO: retrieve progress from ETCD
	return nil
}

func (pd *PartialData) Retrieve() (map[string]interface{}, error) {
	// TODO: if data is colocated in this Serverledge node, we should get data from here
	//  otherwise, retrieve data from ETCD
	return pd.Data, nil
}

func (pd *PartialData) Save() {
	// TODO: save data on ETCD
}

func (pd *PartialData) Purge() {
	// TODO: delete from etcd: all partial data connected to the same request should be deleted, only after the dag is complete.
}

// TODO: We should have a local cache for this data and progress!!!
