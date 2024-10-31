package fc

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/types"
	"github.com/lithammer/shortuuid"

	// "strconv"
	"strings"
)

// ChoiceNode receives one input and produces one result to one of two alternative nodes, based on condition
type ChoiceNode struct {
	Id           DagNodeId
	NodeType     DagNodeType
	BranchId     int
	input        map[string]interface{}
	Alternatives []DagNodeId
	Conditions   []Condition
	FirstMatch   int
}

func NewChoiceNode(conds []Condition) *ChoiceNode {
	return &ChoiceNode{
		Id:       DagNodeId(shortuuid.New()),
		NodeType: Choice,
		//input         make(map[string]interface{}, )
		Conditions:   conds,
		Alternatives: make([]DagNodeId, len(conds)),
		FirstMatch:   -1,
	}
}

// The condition function must be created from the Dag specification in State Function Language or AFCL

func (c *ChoiceNode) Equals(cmp types.Comparable) bool {
	switch cmp.(type) {
	case *ChoiceNode:
		c2 := cmp.(*ChoiceNode)
		if len(c.Conditions) != len(c2.Conditions) || len(c.Alternatives) != len(c2.Alternatives) {
			return false
		}
		for i := 0; i < len(c.Alternatives); i++ {
			if c.Alternatives[i] != c2.Alternatives[i] {
				return false
			}
			if !c.Conditions[i].Equals(c2.Conditions[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// Exec for choice node evaluates the condition
func (c *ChoiceNode) Exec(compRequest *CompositionRequest, params ...map[string]interface{}) (map[string]interface{}, error) {
	t0 := time.Now()
	output := make(map[string]interface{})
	var err error = nil

	if len(params) != 1 {
		return nil, fmt.Errorf("failed to get one input for choice node: received %d inputs", len(params))
	}

	// simply evalutes the Conditions and set the matching one
	for i, condition := range c.Conditions {
		ok, err := condition.Test(params[0])
		if err != nil {
			return nil, fmt.Errorf("error while testing condition: %v", err)
		}
		if ok {
			c.FirstMatch = i
			// the output map should be like the input map!
			output = params[0]
			break
		}
	}
	if c.FirstMatch == -1 {
		err = fmt.Errorf("no condition is met")
		output["error"] = fmt.Sprintf("failed choice node %s - no condition is met", c.Id)
	}
	respAndDuration := time.Now().Sub(t0).Seconds()

	execReport := &function.ExecutionReport{
		Result:         fmt.Sprintf("%v", output),
		ResponseTime:   respAndDuration,
		IsWarmStart:    true, // not in a container
		InitTime:       0,
		OffloadLatency: 0,
		Duration:       respAndDuration,
		SchedAction:    "",
	}

	compRequest.ExecReport.Reports.Set(CreateExecutionReportId(c), execReport)
	return output, err
}

func (c *ChoiceNode) AddOutput(dag *Dag, dagNode DagNodeId) error {

	if len(c.Alternatives) > len(c.Conditions) {
		return errors.New(fmt.Sprintf("there are %d alternatives but %d Conditions", len(c.Alternatives), len(c.Conditions)))
	}
	c.Alternatives = append(c.Alternatives, dagNode)
	if len(c.Alternatives) > len(c.Conditions) {
		return errors.New(fmt.Sprintf("there are %d alternatives but %d Conditions", len(c.Alternatives), len(c.Conditions)))
	}
	return nil
}

func (c *ChoiceNode) CheckInput(input map[string]interface{}) error {
	return nil
}

func (c *ChoiceNode) PrepareOutput(dag *Dag, output map[string]interface{}) error {
	// we should map the output to the input of the node that first matches the condition and not to every alternative
	for _, n := range c.GetNext() {
		dagNode, ok := dag.Find(n)
		if !ok {
			return fmt.Errorf("node not found while preparing output")
		}
		switch nod := dagNode.(type) {
		case *SimpleNode:
			return nod.MapOutput(output)
		}
	}
	return nil
}

// MapOutput transforms the output for the next simpleNode, to make it compatible with its Signature
func (s *ChoiceNode) MapOutput(output map[string]interface{}, sign function.Signature) error {

	// if there are no inputs, we do nothing
	for _, def := range sign.GetInputs() {
		// if output has same name as input, we do not need to change name
		_, present := output[def.Name]
		if present {
			continue
		}
		// find an entry in the output map that successfully checks the type of the InputDefinition
		key, ok := def.FindEntryThatTypeChecks(output)
		if ok {
			// we get the output value
			val := output[key]
			// we remove the output entry ...
			delete(output, key)
			// and replace with the input entry
			output[def.Name] = val
			// save the output map in the input of the node
			//s.inputMutex.Lock()
			//s.input = output
			//s.inputMutex.Unlock()
		} else {
			// otherwise if no one of the entry typechecks we are doomed
			return fmt.Errorf("no output entry input-checks with the next function")
		}
	}
	// if the outputs are more than the needed input, we do nothing
	return nil
}

// GetChoiceBranch returns all node ids of a branch under a choice node; branch number starts from 0
func (c *ChoiceNode) GetChoiceBranch(dag *Dag, branch int) []DagNode {
	branchNodes := make([]DagNode, 0)
	if len(c.Alternatives) <= branch {
		fmt.Printf("fail to get branch %d\n", branch)
		return branchNodes
	}
	node := c.Alternatives[branch]
	return VisitDag(dag, node, branchNodes, true)
}

// GetNodesToSkip skips all node that are in a branch that will not be executed.
// If a skipped branch contains one or more node that is used by the current branch, the node,
// should NOT be skipped (Tested in TestParsingChoiceDagWithDataTestExpr)
func (c *ChoiceNode) GetNodesToSkip(dag *Dag) []DagNode {
	nodesToSkip := make([]DagNode, 0)
	if c.FirstMatch == -1 || c.FirstMatch >= len(c.Alternatives) {
		return nodesToSkip
	}

	nodesToNotSkip := c.GetChoiceBranch(dag, c.FirstMatch)
	for i := 0; i < len(c.Alternatives); i++ {
		if i == c.FirstMatch {
			continue
		}
		branchNodes := c.GetChoiceBranch(dag, i)
		for _, node := range branchNodes {
			shouldBeSkipped := true
			for _, nodeToNotSkip := range nodesToNotSkip {
				if node.Equals(nodeToNotSkip) {
					shouldBeSkipped = false
					break
				}
			}
			if shouldBeSkipped {
				nodesToSkip = append(nodesToSkip, node)
			}
		}
	}
	return nodesToSkip
}

func (c *ChoiceNode) GetNext() []DagNodeId {
	// you should have called exec before calling GetNext
	if c.FirstMatch >= len(c.Alternatives) {
		panic("there aren't sufficient alternatives!")
	}

	if c.FirstMatch < 0 {
		panic("first match cannot be less then 0. You should call Exec() before GetNext()")
	}

	return []DagNodeId{c.Alternatives[c.FirstMatch]}
}

func (c *ChoiceNode) Width() int {
	return len(c.Alternatives)
}

func (c *ChoiceNode) Name() string {
	n := len(c.Conditions)

	if n%2 == 0 {
		// se n =10 : -9 ---------
		// se n = 8 : -7 -------
		// se n = 6 : -5
		// se n = 4 : -3
		// se n = 2 : -1
		// [Simple|Simple|Simple|Simple|Simple|Simple|Simple|Simple|Simple|Simple]
		return strings.Repeat("-", 4*(n-1)-n/2) + "Choice" + strings.Repeat("-", 3*(n-1)+n/2)
	} else {
		pad := "-------"
		return strings.Repeat(pad, int(math.Max(float64(n/2), 0.))) + "Choice" + strings.Repeat(pad, int(math.Max(float64(n/2), 0.)))
	}
}

func (c *ChoiceNode) setBranchId(number int) {
	c.BranchId = number
}

func (c *ChoiceNode) GetBranchId() int {
	return c.BranchId
}

func (c *ChoiceNode) String() string {
	conditions := "<"
	for i, condFn := range c.Conditions {
		conditions += condFn.String()
		if i != len(c.Conditions) {
			conditions += " | "
		}
	}
	conditions += ">"
	return fmt.Sprintf("[ChoiceNode(%d): %s] ", len(c.Alternatives), conditions)
}

func (c *ChoiceNode) GetId() DagNodeId {
	return c.Id
}

func (c *ChoiceNode) GetNodeType() DagNodeType {
	return c.NodeType
}
