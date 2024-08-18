package asl

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/types"
	"strings"
)

type ChoiceRule interface {
	types.Comparable
	fmt.Stringer
	// IsBooleanExpression return true if the ChoiceRule is a BooleanExpression, otherwise false when it is a DataTestExpression
	IsBooleanExpression() bool
}

// BooleanExpression is a ChoiceRule that is parsable from combination of And, Or and Not json objects
type BooleanExpression struct {
	And  []*ChoiceRule
	Or   []*ChoiceRule
	Not  *ChoiceRule
	Next string // Necessary. The value should match a state name in the StateMachine
}

func (b *BooleanExpression) IsBooleanExpression() bool {
	return true
}

func ParseBooleanExpr(jsonRule []byte) (*BooleanExpression, error) {
	return &BooleanExpression{}, nil
}

// DataTestExpression is a ChoiceRule that is parsable from a Variable, a ComparisonOperator and has a Next field.
type DataTestExpression struct {
	Variable           string
	ComparisonOperator *ComparisonOperator // FIXME: parse into a fc.Condition, but leave it a string (or you'll have an import cycle)
	Next               string              // Necessary. The value should match a state name in the StateMachine
}

func (d *DataTestExpression) IsBooleanExpression() bool {
	return false
}

func (d *DataTestExpression) String() string {
	str := "\n\t\t\t\t{"

	if d.Variable != "" {
		str += fmt.Sprintf("\n\t\t\t\t\tVariable: %s\n", d.Variable)
	}
	if d.ComparisonOperator != nil {
		str += d.ComparisonOperator.String()
	}
	if d.Next != "" {
		str += fmt.Sprintf("\t\t\t\t\tNext: %s", d.Next)
	}
	return str + "\n\t\t\t\t}"
}

func (d *DataTestExpression) Equals(cmp types.Comparable) bool {
	d2 := cmp.(*DataTestExpression)
	return d.Next == d2.Next &&
		d.ComparisonOperator.Equals(d2.ComparisonOperator) &&
		d.Variable == d2.Variable
}

func ParseDataTestExpr(jsonRule []byte) (*DataTestExpression, error) {
	variable, err := JsonExtractString(jsonRule, "Variable")
	if err != nil {
		return nil, err
	}
	next, err := JsonExtractString(jsonRule, "Next")
	if err != nil {
		return nil, err
	}

	comparisonOperator := ComparisonOperator{}
	for i, comparator := range PossibleComparators {
		comparatorValue, errComp := JsonExtractString(jsonRule, string(comparator))
		if errComp != nil {
			if i == len(PossibleComparators)-1 {
				return nil, fmt.Errorf("invalid data test expression comparator. It can be one of the following: StringEquals, StringEqualsPath, StringLessThan, StringLessThanPath, StringGreaterThan, StringGreaterThanPath, StringLessThanEquals, StringLessThanEqualsPath, StringGreaterThanEquals, StringGreaterThanEqualsPath, StringMatches, NumericEqualsPath, NumericLessThan, NumericLessThanPath, NumericGreaterThan, NumericGreaterThanPath, NumericLessThanEquals, NumericLessThanEqualsPath, NumericGreaterThanEquals, NumericGreaterThanEqualsPath, BooleanEquals, BooleanEqualsPath, TimestampEquals, TimestampEqualsPath, TimestampLessThan, TimestampLessThanPath, TimestampGreaterThan, TimestampGreaterThanPath, TimestampLessThanEquals, TimestampLessThanEqualsPath, TimestampGreaterThanEquals, TimestampGreaterThanEqualsPath, IsNull, IsPresent, IsNumeric, IsString, IsBoolean, IsTimestamp")
			}
			continue
		}
		// comparator kind field
		comparisonOperator.Kind = comparator
		comparatorString := string(comparator)
		// comparator datatype field
		if strings.HasPrefix(comparatorString, "String") {
			comparisonOperator.DataType = StringComparator
		} else if strings.HasPrefix(comparatorString, "Numeric") {
			comparisonOperator.DataType = NumericComparator
		} else if strings.HasPrefix(comparatorString, "Timestamp") {
			comparisonOperator.DataType = TimestampComparator
		} else if strings.HasPrefix(comparatorString, "Boolean") || strings.HasPrefix(comparatorString, "Is") {
			comparisonOperator.DataType = BooleanComparator
		} else {
			return nil, fmt.Errorf("invalid comparator: %s", comparator)
		}
		// comparator operand field
		comparisonOperator.Operand = comparatorValue
		// we have found a valid comparator, so we exit the for loop
		break
	}

	// check that there is only one comparisonOperator
	num := JsonNumberOfKeys(jsonRule)
	if num != 3 {
		return nil, fmt.Errorf("found %d keys. There are'nt exactly 3 keys in this JsonRule: %s\n", num, jsonRule)
	}

	return &DataTestExpression{
		Variable:           variable,
		ComparisonOperator: &comparisonOperator,
		Next:               next,
	}, nil
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
