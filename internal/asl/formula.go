package asl

import (
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/grussorusso/serverledge/internal/types"
)

type Formula interface {
	types.Comparable
	fmt.Stringer
	GetFormulaType() string
}

type AndFormula struct {
	And []ChoiceRule
}

func (a *AndFormula) Equals(cmp types.Comparable) bool {
	otherAnd, ok := cmp.(*AndFormula)
	if !ok {
		return false
	}
	for i, choiceRule := range a.And {
		if !choiceRule.Equals(otherAnd.And[i]) {
			return false
		}
	}

	return true
}

func (a *AndFormula) String() string {
	str := "\t\t\t\t\t\tAnd: ["

	for i, choiceRule := range a.And {
		str += "\n\t\t\t\t\t\t\t" + choiceRule.String()
		if i != len(a.And)-1 {
			str += ","
		}
	}

	str += "]"

	return str
}

func (a *AndFormula) GetFormulaType() string {
	return "And"
}

func ParseAnd(jsonBytes []byte) (*AndFormula, error) {
	andJson, err := JsonExtract(jsonBytes, "And")
	if err != nil {
		return nil, fmt.Errorf("failed to parse and formula: %v", err)
	}
	andArray := make([]ChoiceRule, 0)
	_, err = jsonparser.ArrayEach(andJson, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		rule, err := ParseRule(andJson)
		if err != nil {
			return
		}
		andArray = append(andArray, rule)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse and formula: %v", err)
	}

	return &AndFormula{And: andArray}, nil
}

type OrFormula struct {
	Or []ChoiceRule
}

func ParseOr(jsonBytes []byte) (*OrFormula, error) {
	orRule, err := JsonExtract(jsonBytes, "Or")
	if err != nil {
		return nil, fmt.Errorf("failed to parse Or formula: %v", err)
	}
	orArray := make([]ChoiceRule, 0)
	_, err = jsonparser.ArrayEach(orRule, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		rule, err := ParseRule(orRule)
		if err != nil {
			return
		}
		orArray = append(orArray, rule)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse Or formula: %v", err)
	}

	return &OrFormula{Or: orArray}, nil
}

func (o *OrFormula) Equals(cmp types.Comparable) bool {
	otherOr, ok := cmp.(*OrFormula)
	if !ok {
		return false
	}
	for i, choiceRule := range o.Or {
		if !choiceRule.Equals(otherOr.Or[i]) {
			return false
		}
	}

	return true
}

func (o *OrFormula) String() string {
	str := "\t\t\t\t\t\tOr: ["

	for i, choiceRule := range o.Or {
		str += "\n\t\t\t\t\t\t\t" + choiceRule.String()
		if i != len(o.Or)-1 {
			str += ","
		}
	}

	str += "]"

	return str
}

func (o *OrFormula) GetFormulaType() string {
	return "Or"
}

type NotFormula struct {
	Not ChoiceRule
}

func ParseNot(jsonBytes []byte) (*NotFormula, error) {
	notJson, err := JsonExtract(jsonBytes, "Not")
	if err != nil {
		return nil, fmt.Errorf("failed to parse Not formula: %v", err)
	}
	notRule, err := ParseRule(notJson)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Not formula: %v", err)
	}
	return &NotFormula{Not: notRule}, nil
}

func (n *NotFormula) Equals(cmp types.Comparable) bool {
	otherNot, ok := cmp.(*NotFormula)
	if !ok {
		return false
	}
	return otherNot.Not.Equals(n.Not)
}

func (n *NotFormula) String() string {
	return fmt.Sprintf("\t\t\t\t\t\tNot: %s", n.Not.String())
}

func (n *NotFormula) GetFormulaType() string {
	return "Not"
}
