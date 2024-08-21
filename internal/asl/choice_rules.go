package asl

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
	"strconv"
	"strings"
)

type RuleType int

const (
	BooleanExpr = iota
	DataTestExpr
	TestExpr
)

type ChoiceRule interface {
	types.Comparable
	fmt.Stringer
	Validatable
	// IsBooleanExpression return true if the ChoiceRule is a BooleanExpression, otherwise false when it is a DataTestExpression
	GetRuleType() RuleType
}

func ParseRule(json []byte) (ChoiceRule, error) {
	// detect if it is a boolean or dataTest expression
	if JsonHasAllKeys(json, "Variable", "Next") && JsonNumberOfKeys(json) == 3 {
		return ParseDataTestExpr(json)
	} else if JsonHasOneKey(json, "And", "Or", "Not") {
		return ParseBooleanExpr(json)
	} else {
		return nil, fmt.Errorf("invalid choice rule: %s", string(json))
	}
}

// BooleanExpression is a ChoiceRule that is parsable from combination of And, Or and Not json objects
type BooleanExpression struct {
	Formula Formula
	Next    string // Necessary. The value should match a state name in the StateMachine
}

func (b *BooleanExpression) Equals(cmp types.Comparable) bool {
	b2, ok := cmp.(*BooleanExpression)
	if !ok {
		return false
	}
	return b.Next == b2.Next && b.Formula.Equals(b2.Formula)
}

func (b *BooleanExpression) String() string {
	return "\t\t\t\t\t" + b.Formula.GetFormulaType() + ": {\n" +
		b.Formula.String() +
		"\n\t\t\t\t\tNext: " + b.Next +
		"\n\t\t\t\t\t}"
}

func (b *BooleanExpression) GetRuleType() RuleType {
	return BooleanExpr
}

func ParseBooleanExpr(jsonExpression []byte) (*BooleanExpression, error) {
	next, err := JsonExtractString(jsonExpression, "Next")
	if err != nil {
		return nil, fmt.Errorf("boolean expression doesn't have a Next json field")
	}
	if JsonHasKey(jsonExpression, "And") {
		andFormula, err2 := ParseAnd(jsonExpression)
		if err2 != nil {
			return nil, err2
		}
		return &BooleanExpression{Formula: andFormula, Next: next}, nil
	} else if JsonHasKey(jsonExpression, "Or") {
		orFormula, err2 := ParseOr(jsonExpression)
		if err2 != nil {
			return nil, err2
		}
		return &BooleanExpression{Formula: orFormula, Next: next}, nil
	} else if JsonHasKey(jsonExpression, "Not") {
		notFormula, err2 := ParseNot(jsonExpression)
		if err2 != nil {
			return nil, err2
		}
		return &BooleanExpression{Formula: notFormula, Next: next}, nil
	}
	return nil, fmt.Errorf("invalid boolean expression: %s", string(jsonExpression))
}

func (b *BooleanExpression) Validate(stateNames []string) error {
	//TODO implement me
	return nil
}

// TestExpression is used in both DataTestExpression and BooleanExpression and represents a condition
type TestExpression struct {
	Variable           string
	ComparisonOperator *ComparisonOperator // FIXME: parse into a fc.Condition, but leave it a string (or you'll have an import cycle)
}

func (t *TestExpression) Validate(stateNames []string) error {
	return nil
}

func (t *TestExpression) String() string {
	str := "\n\t\t\t\t{"

	if t.Variable != "" {
		str += fmt.Sprintf("\n\t\t\t\t\tVariable: %s\n", t.Variable)
	}
	if t.ComparisonOperator != nil {
		str += t.ComparisonOperator.String()
	}
	return str + "\n\t\t\t\t}"
}

func (t *TestExpression) GetRuleType() RuleType {
	return TestExpr
}

func (t *TestExpression) Equals(cmp types.Comparable) bool {
	d2 := cmp.(*TestExpression)
	return t.Variable == d2.Variable &&
		t.ComparisonOperator.Equals(d2.ComparisonOperator)
}

func ParseTestExpr(jsonRule []byte) (*TestExpression, error) {
	variable, err := JsonExtractString(jsonRule, "Variable")
	if err != nil {
		return nil, err
	}

	comparisonOperator := ComparisonOperator{}
	for i, comparator := range PossibleComparators {
		comparatorValue, errComp := JsonExtractString(jsonRule, string(comparator))
		if errComp != nil {
			if i == len(PossibleComparators)-1 {
				return nil, fmt.Errorf("invalid test expression comparator. It can be one of the following: StringEquals, StringEqualsPath, StringLessThan, StringLessThanPath, StringGreaterThan, StringGreaterThanPath, StringLessThanEquals, StringLessThanEqualsPath, StringGreaterThanEquals, StringGreaterThanEqualsPath, StringMatches, NumericEqualsPath, NumericLessThan, NumericLessThanPath, NumericGreaterThan, NumericGreaterThanPath, NumericLessThanEquals, NumericLessThanEqualsPath, NumericGreaterThanEquals, NumericGreaterThanEqualsPath, BooleanEquals, BooleanEqualsPath, TimestampEquals, TimestampEqualsPath, TimestampLessThan, TimestampLessThanPath, TimestampGreaterThan, TimestampGreaterThanPath, TimestampLessThanEquals, TimestampLessThanEqualsPath, TimestampGreaterThanEquals, TimestampGreaterThanEqualsPath, IsNull, IsPresent, IsNumeric, IsString, IsBoolean, IsTimestamp")
			}
			continue
		}
		// comparator kind field
		comparisonOperator.Kind = comparator
		comparatorString := string(comparator)
		// comparator datatype and operand fields
		if strings.HasPrefix(comparatorString, "String") {
			comparisonOperator.DataType = StringComparator
			comparisonOperator.Operand = comparatorValue
		} else if strings.HasPrefix(comparatorString, "Numeric") {
			comparisonOperator.DataType = NumericComparator
			comparisonOperator.Operand, err = strconv.Atoi(comparatorValue)
			if err != nil {
				return nil, fmt.Errorf("failed to convert to int the value %s: %v", comparatorValue, err)
			}
		} else if strings.HasPrefix(comparatorString, "Timestamp") {
			comparisonOperator.DataType = TimestampComparator
			comparisonOperator.Operand = comparatorValue
		} else if strings.HasPrefix(comparatorString, "Boolean") || strings.HasPrefix(comparatorString, "Is") {
			comparisonOperator.DataType = BooleanComparator
			comparisonOperator.Operand, err = strconv.ParseBool(comparatorValue)
			if err != nil {
				return nil, fmt.Errorf("failed to convert to bool the value %s: %v", comparatorValue, err)
			}
		} else {
			return nil, fmt.Errorf("invalid comparator: %s", comparator)
		}

		// we have found a valid comparator, so we exit the for loop
		break
	}
	return &TestExpression{
		Variable:           variable,
		ComparisonOperator: &comparisonOperator,
	}, nil
}

// DataTestExpression is a ChoiceRule that is parsable from a Variable, a ComparisonOperator and has a Next field.
type DataTestExpression struct {
	Test *TestExpression
	Next string // Necessary. The value should match a state name in the StateMachine
}

func (d *DataTestExpression) GetRuleType() RuleType {
	return DataTestExpr
}

func (d *DataTestExpression) String() string {
	str := "\n\t\t\t\t{"

	if d.Test.Variable != "" {
		str += fmt.Sprintf("\n\t\t\t\t\tVariable: %s\n", d.Test.Variable)
	}
	if d.Test.ComparisonOperator != nil {
		str += d.Test.ComparisonOperator.String()
	}
	if d.Next != "" {
		str += fmt.Sprintf("\t\t\t\t\tNext: %s", d.Next)
	}
	return str + "\n\t\t\t\t}"
}

func (d *DataTestExpression) Equals(cmp types.Comparable) bool {
	d2 := cmp.(*DataTestExpression)
	return d.Test.Equals(d2.Test) && d.Next == d2.Next
}

func ParseDataTestExpr(jsonRule []byte) (*DataTestExpression, error) {
	next, err := JsonExtractString(jsonRule, "Next")
	if err != nil {
		return nil, err
	}

	testExpr, err := ParseTestExpr(jsonRule)
	if err != nil {
		return nil, fmt.Errorf("failed to parse test expression %s\n%v", jsonRule, err)
	}

	return &DataTestExpression{
		Test: testExpr,
		Next: next,
	}, nil
}

func (d *DataTestExpression) Validate(stateNames []string) error {
	// TODO: implement me.
	return nil
}

type ComparisonOperator struct {
	Kind     ComparisonOperatorKind
	DataType ComparisonOperatorDataType
	Operand  interface{}
}

func (co *ComparisonOperator) String() string {
	return fmt.Sprintf("\t\t\t\t\t%s: %v\n", co.Kind, co.Operand)
}

func (co *ComparisonOperator) Equals(co2 *ComparisonOperator) bool {
	return co.Kind == co2.Kind && co.DataType == co2.DataType && co.Operand == co2.Operand
}

var PossibleComparators = []ComparisonOperatorKind{
	StringEquals, StringEqualsPath, StringLessThan, StringLessThanPath, StringGreaterThan, StringGreaterThanPath, StringLessThanEquals, StringLessThanEqualsPath, StringGreaterThanEquals, StringGreaterThanEqualsPath, StringMatches, NumericEquals, NumericEqualsPath, NumericLessThan, NumericLessThanPath, NumericGreaterThan, NumericGreaterThanPath, NumericLessThanEquals, NumericLessThanEqualsPath, NumericGreaterThanEquals, NumericGreaterThanEqualsPath, BooleanEquals, BooleanEqualsPath, TimestampEquals, TimestampEqualsPath, TimestampLessThan, TimestampLessThanPath, TimestampGreaterThan, TimestampGreaterThanPath, TimestampLessThanEquals, TimestampLessThanEqualsPath, TimestampGreaterThanEquals, TimestampGreaterThanEqualsPath, IsNull, IsPresent, IsNumeric, IsString, IsBoolean, IsTimestamp,
}

type ComparisonOperatorKind string

const (
	StringEquals                   ComparisonOperatorKind = "StringEquals"
	StringEqualsPath               ComparisonOperatorKind = "StringEqualsPath"
	StringLessThan                 ComparisonOperatorKind = "StringLessThan"
	StringLessThanPath             ComparisonOperatorKind = "StringLessThanPath"
	StringGreaterThan              ComparisonOperatorKind = "StringGreaterThan"
	StringGreaterThanPath          ComparisonOperatorKind = "StringGreaterThanPath"
	StringLessThanEquals           ComparisonOperatorKind = "StringLessThanEquals"
	StringLessThanEqualsPath       ComparisonOperatorKind = "StringLessThanEqualsPath"
	StringGreaterThanEquals        ComparisonOperatorKind = "StringGreaterThanEquals"
	StringGreaterThanEqualsPath    ComparisonOperatorKind = "StringGreaterThanEqualsPath"
	StringMatches                  ComparisonOperatorKind = "StringMatches" // StringMatches The value MUST be a StringComparator which MAY contain one or more "*" characters. The expression yields true if the data value selected by the Variable Path matches the value, where "*" in the value matches zero or more characters. Thus, foo*.log matches foo23.log, *.log matches zebra.log, and foo*.* matches foobar.zebra. No characters other than "*" have any special meaning during matching.	NumericEquals                  ComparisonOperatorKind = "NumericEquals"
	NumericEquals                  ComparisonOperatorKind = "NumericEquals"
	NumericEqualsPath              ComparisonOperatorKind = "NumericEqualsPath"
	NumericLessThan                ComparisonOperatorKind = "NumericLessThan"
	NumericLessThanPath            ComparisonOperatorKind = "NumericLessThanPath"
	NumericGreaterThan             ComparisonOperatorKind = "NumericGreaterThan"
	NumericGreaterThanPath         ComparisonOperatorKind = "NumericGreaterThanPath"
	NumericLessThanEquals          ComparisonOperatorKind = "NumericLessThanEquals"
	NumericLessThanEqualsPath      ComparisonOperatorKind = "NumericLessThanEqualsPath"
	NumericGreaterThanEquals       ComparisonOperatorKind = "NumericGreaterThanEquals"
	NumericGreaterThanEqualsPath   ComparisonOperatorKind = "NumericGreaterThanEqualsPath"
	BooleanEquals                  ComparisonOperatorKind = "BooleanEquals"
	BooleanEqualsPath              ComparisonOperatorKind = "BooleanEqualsPath"
	TimestampEquals                ComparisonOperatorKind = "TimestampEquals"
	TimestampEqualsPath            ComparisonOperatorKind = "TimestampEqualsPath"
	TimestampLessThan              ComparisonOperatorKind = "TimestampLessThan"
	TimestampLessThanPath          ComparisonOperatorKind = "TimestampLessThanPath"
	TimestampGreaterThan           ComparisonOperatorKind = "TimestampGreaterThan"
	TimestampGreaterThanPath       ComparisonOperatorKind = "TimestampGreaterThanPath"
	TimestampLessThanEquals        ComparisonOperatorKind = "TimestampLessThanEquals"
	TimestampLessThanEqualsPath    ComparisonOperatorKind = "TimestampLessThanEqualsPath"
	TimestampGreaterThanEquals     ComparisonOperatorKind = "TimestampGreaterThanEquals"
	TimestampGreaterThanEqualsPath ComparisonOperatorKind = "TimestampGreaterThanEqualsPath"
	IsNull                         ComparisonOperatorKind = "IsNull"    // IsNull This means the value is the built-in JSON literal null.
	IsPresent                      ComparisonOperatorKind = "IsPresent" // IsPresent in this case, if the Variable-field Path fails to match anything in the input no exception is thrown and the Choice Rule just returns false.
	IsNumeric                      ComparisonOperatorKind = "IsNumeric"
	IsString                       ComparisonOperatorKind = "IsString"
	IsBoolean                      ComparisonOperatorKind = "IsBoolean"
	IsTimestamp                    ComparisonOperatorKind = "IsTimestamp"
)

type ComparisonOperatorDataType int

const (
	StringComparator = iota
	NumericComparator
	TimestampComparator
	BooleanComparator
)
