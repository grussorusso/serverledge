package fc

import (
	"errors"
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
	"math"
	// "strconv"
	"strings"
)

// ChoiceNode receives one input and produces one result to one of two alternative nodes, based on condition
type ChoiceNode struct {
	input map[string]interface{}
	// InputFrom    DagNode
	Alternatives []DagNode
	Conditions   []Condition
	firstMatch   int
}

func NewChoiceNode(conds []Condition) *ChoiceNode {
	return &ChoiceNode{
		Conditions:   conds,
		Alternatives: make([]DagNode, len(conds)),
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
			//if c.Conditions[i] != c2.Conditions[i] {
			//	return false // TODO: how to compare function Conditions?
			//}
		}
		return true
	default:
		return false
	}
}

// Exec for choice node evaluates the condition
func (c *ChoiceNode) Exec() (map[string]interface{}, error) {
	// simply evalutes the Conditions and set the matching one
	for i, condition := range c.Conditions {
		ok, err := condition.Test()
		if err != nil {
			return nil, fmt.Errorf("error while testing condition: %v", err)
		}
		if ok {
			c.firstMatch = i
			// the output map should be like the input map!
			return c.input, nil
		}
	}
	return nil, fmt.Errorf("no condition is met")
}

func (c *ChoiceNode) AddInput(dagNode DagNode) error {
	//if c.InputFrom != nil {
	//	return errors.New("input already present in node")
	//}
	//
	//c.InputFrom = dagNode
	return nil
}

// TODO: thats a bit useless
func (c *ChoiceNode) AddCondition(condition Condition) {
	c.Conditions = append(c.Conditions, condition)
}

func (c *ChoiceNode) AddOutput(dagNode DagNode) error {

	if len(c.Alternatives) > len(c.Conditions) {
		return errors.New(fmt.Sprintf("there are %d alternatives but %d Conditions", len(c.Alternatives), len(c.Conditions)))
	}
	c.Alternatives = append(c.Alternatives, dagNode)
	if len(c.Alternatives) > len(c.Conditions) {
		return errors.New(fmt.Sprintf("there are %d alternatives but %d Conditions", len(c.Alternatives), len(c.Conditions)))
	}
	return nil
}

func (c *ChoiceNode) ReceiveInput(input map[string]interface{}) error {
	c.input = input
	return nil
}

func (c *ChoiceNode) PrepareOutput(output map[string]interface{}) error {
	// we should map the output to the input of the node that first matches the condition and not to every alternative
	for _, n := range c.GetNext() {
		switch n.(type) {
		case *SimpleNode:
			return n.(*SimpleNode).MapOutput(output)
		}
	}
	return nil
}

func (c *ChoiceNode) GetNext() []DagNode {
	// you should have called exec before calling GetNext
	if c.firstMatch >= len(c.Alternatives) {
		panic("there aren't sufficient alternatives!")
	}

	if c.firstMatch < 0 {
		panic("first match cannot be less then 0")
	}
	next := make([]DagNode, 1)
	next[0] = c.Alternatives[c.firstMatch]
	return next
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

func (c *ChoiceNode) ToString() string {
	conditions := "<"
	// FIXME: cannot represent functions
	//for i, condFn := range c.Conditions {
	//	conditions += strconv.FormatBool(condFn())
	//	if i != len(c.Conditions) {
	//		conditions += " | "
	//	}
	//}
	conditions += ">"
	return fmt.Sprintf("[ChoiceNode(%d): %s] ", c.Alternatives, conditions)
}
