package asl

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
)

type ChoiceState struct {
	Type       StateType     // Necessary
	Choices    []*ChoiceRule // Necessary. All ChoiceRule must be State Machine with an end, like fc.ChoiceNode(s).
	InputPath  Path          // Optional
	OutputPath Path          // Optional
	Default    string        // Optional, but to avoid errors it is highly recommended
}

// TODO
func (c *ChoiceState) ParseFrom(jsonData []byte) (State, error) {
	/*choices, _, _, errChoice := jsonparser.Get(value, "Choices")
	if errChoice != nil {
		return errChoice
	}
	matches := make([]ChoiceRule, 0)

	_, errArr := jsonparser.ArrayEach(choices, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		matchVariable, _, _, errMatch := jsonparser.Get(value, "Variable")
		if errMatch != nil {
			return
		}
		matchCondition, _, _, errMatch := jsonparser.Get(value, "NumericEquals") // TODO: only checks NumericEquals
		if errMatch != nil {
			return
		}
		num, errNum := strconv.Atoi(string(matchCondition))
		if errNum != nil {
			return
		}
		matchNext, _, _, errMatch := jsonparser.Get(value, "Next")
		if errMatch != nil {
			return
		}
		m := ChoiceRule{
			Variable:  string(matchVariable),
			Operation: NewEqCondition(NewParam(string(matchVariable)), NewValue(num)),
			Next:      string(matchNext),
		}
		matches = append(matches, m)
	})
	if errArr != nil {
		return errArr
	}

	defaultMatch, _, _, errChoice := jsonparser.Get(value, "Default")
	if errChoice != nil {
		return err
	}*/
	return nil, nil
}

func NewEmptyChoice() *ChoiceState {
	return &ChoiceState{
		Type:       Choice,
		Choices:    []*ChoiceRule{},
		InputPath:  "",
		OutputPath: "",
		Default:    "",
	}
}

func (c *ChoiceState) GetType() StateType {
	return Choice
}

// GetNext for ChoiceState returns the Default branch instead of next
func (c *ChoiceState) GetNext() (string, bool) {
	if c.Default != "" {
		return c.Default, true
	}
	return "", false
}

// IsEndState always returns false for a ChoiceState, because it is never a terminal state.
func (c *ChoiceState) IsEndState() bool {
	return false
}

func (c *ChoiceState) Equals(cmp types.Comparable) bool {
	c2 := cmp.(*ChoiceState)

	for _, c1 := range c.Choices {
		if !c1.Equals(c2) {
			return false
		}
	}

	return c.Type == c2.Type &&
		c.InputPath == c2.InputPath &&
		c.OutputPath == c2.OutputPath &&
		c.Default == c2.Default
}

// FIXME: improve
func (c *ChoiceState) String() string {
	str := fmt.Sprint("{",
		"\n\t\t\t\tType: ", c.Type,
		"\n\t\t\t\tDefault: ", c.Default,
		"\n\t\t\t\tChoices: [")
	for i, c1 := range c.Choices {
		str += c1.String()
		if i < len(c.Choices)-1 {
			str += ","
		}
	}
	str += "]"

	if c.InputPath != "" {
		str += fmt.Sprintf("\t\t\t\tInputPath: %s\n", c.InputPath)
	}
	if c.OutputPath != "" {
		str += fmt.Sprintf("\t\t\t\tOutputPath: %s\n", c.OutputPath)
	}
	str += "\t\t\t}\n"
	return str
}

type ChoiceRule struct {
	Variable  string
	Operation string // FIXME: come up with a better type (Do not use fc.Condition, or you will have an import cycle)
	Next      string
}

// FIXME: improve
func (cr *ChoiceRule) String() string {
	str := "\n\t\t\t\t\t{"

	if cr.Variable != "" {
		str += fmt.Sprintf("\t\t\t\t\t\tVariable: %s\n", cr.Variable)
	}
	if cr.Operation != "" {
		str += fmt.Sprintf("\t\t\t\t\t\tOperation: %s\n", cr.Operation)
	}
	if cr.Next != "" {
		str += fmt.Sprintf("\t\t\t\t\t\tNext: %s\n", cr.Next)
	}
	return str + "\n\t\t\t\t\t}"
}

func (cr *ChoiceRule) Equals(cmp types.Comparable) bool {
	cr2 := cmp.(*ChoiceRule)
	return cr.Next == cr2.Next &&
		cr.Operation == cr2.Operation &&
		cr.Variable == cr2.Variable
}
