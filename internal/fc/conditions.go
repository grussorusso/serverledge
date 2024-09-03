package fc

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/utils"
	"regexp"
	"strconv"
	"strings"
)

type Predicate struct {
	Root Condition
}

type Condition struct {
	Type CondEnum      `json:"Type"` // Type of the condition
	Find []bool        `json:"Find"` // Find if true, the corresponding Op value should be read from parameters
	Op   []interface{} `json:"Op"`   // Op is the list of operand of the condition. An operand can be a constant value or a parameter (a string key that will be used to index the parameter map)
	Sub  []Condition   `json:"Sub"`  // Sub is a SubCondition List. Useful for Type And, Or and Not
}

type CondEnum string

const (
	And           CondEnum = "And"
	Or            CondEnum = "Or"
	Not           CondEnum = "Not"
	Const         CondEnum = "Const"
	Eq            CondEnum = "Eq"        // also works for strings
	Diff          CondEnum = "Diff"      // also works for strings
	Greater       CondEnum = "Greater"   // also works for strings
	Smaller       CondEnum = "Smaller"   // also works for strings
	Empty         CondEnum = "Empty"     // for collections
	IsNumeric     CondEnum = "IsNumeric" // for collections
	StringMatches CondEnum = "StringMatches"
)

func (p Predicate) Test(input map[string]interface{}) bool {
	ok, err := p.Root.Test(input)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
	}
	return ok
}

func (p Predicate) LogicString() string {
	return p.Root.String()
}

func (c Condition) String() string {
	switch c.Type {
	case And:
		str := "("
		for i, condition := range c.Sub {
			str += condition.String()
			if i != len(c.Sub)-1 {
				str += " && "
			}
		}
		str += ")"
		return str
	case Or:
		str := "("
		for i, condition := range c.Sub {
			str += fmt.Sprintf("%s", condition.String())
			if i != len(c.Sub)-1 {
				str += " || "
			}
		}
		str += ")"
		return str
	case Not:
		return fmt.Sprintf("!(%s)", c.Sub[0].String())
	case Const:
		if len(c.Op) == 0 {
			return "?"
		}
		return fmt.Sprintf("%v", c.Op[0])
	case Eq:
		if len(c.Op) == 1 {
			return fmt.Sprintf("%v == ?", c.Op[0])
		} else if len(c.Op) == 0 {
			return "? == ?"
		}
		return fmt.Sprintf("%v == %v", c.Op[0], c.Op[1])
	case Diff:
		if len(c.Op) == 1 {
			return fmt.Sprintf("%v != ?", c.Op[0])
		} else if len(c.Op) == 0 {
			return "? != ?"
		}
		return fmt.Sprintf("%v != %v", c.Op[0], c.Op[1])
	case Greater:
		if len(c.Op) == 1 {
			return fmt.Sprintf("%v > ?", c.Op[0])
		} else if len(c.Op) == 0 {
			return "? > ?"
		}
		return fmt.Sprintf("%v > %v", c.Op[0], c.Op[1])
	case Smaller:
		if len(c.Op) == 1 {
			return fmt.Sprintf("%v < ?", c.Op[0])
		} else if len(c.Op) == 0 {
			return "? < ?"
		}
		return fmt.Sprintf("%v < %v", c.Op[0], c.Op[1])
	case Empty:
		return "empty input"
	case IsNumeric:
		return fmt.Sprintf("IsNumeric(%v)", c.Op[0])
	case StringMatches:
		return fmt.Sprintf("StringMatches(%s,%s)", c.Op[0], c.Op[1])
	default:
		return ""
	}
}

func (p Predicate) Print() {
	fmt.Println(p.LogicString())
}

// findInputs retrieves all the values from the operands.
// If it is a Parameter, finds it in the Op slice and then appends its value to the returned slice.
// If it is a Value, directly adds it to the returned slice
func (c Condition) findInputs(input map[string]interface{}) ([]interface{}, error) {
	ops := make([]interface{}, 0)
	if input == nil {
		return c.Op, nil
	}
	if len(c.Op) != len(c.Find) {
		return nil, fmt.Errorf("size of operand (%d) is different from size of searchable operands array (%d)", len(c.Op), len(c.Find))
	}
	for i := 0; i < len(c.Op); i++ {
		if !c.Find[i] {
			ops = append(ops, c.Op[i])
		} else {
			opStr, ok := c.Op[i].(string)
			if !ok {
				return nil, fmt.Errorf("input name is not a string")
			}
			value, found := input[opStr]
			if !found {
				ops = append(ops, nil)
			} else {
				ops = append(ops, value)
			}
		}
	}
	return ops, nil
}

func (c Condition) Test(input map[string]interface{}) (bool, error) {
	switch c.Type {
	case And:
		and := true
		for _, condition := range c.Sub {
			ok, err := condition.Test(input)
			if err != nil {
				return false, err
			}
			and = and && ok
		}
		return and, nil
	case Or:
		or := false
		for _, condition := range c.Sub {
			ok, err := condition.Test(input)
			if err != nil {
				return false, err
			}
			or = or || ok
		}
		return or, nil
	case Not:
		if len(c.Sub) != 1 {
			return false, fmt.Errorf("you need exactly one condition to check if it is Not satisfied")
		}

		test, err := c.Sub[0].Test(input)
		return !test, err
	case Const:
		if len(c.Op) == 0 {
			return false, fmt.Errorf("you need at least one const operand")
		}
		dataType := function.Bool{}
		ops, err := c.findInputs(input)
		if err != nil {
			return false, err
		}
		return dataType.Convert(ops[0])
	case Eq:
		if len(c.Op) <= 1 {
			return false, fmt.Errorf("you need at least two operands to check equality")
		}
		ops, err := c.findInputs(input)
		if err != nil {
			return false, err
		}
		for i := 0; i < len(ops)-1; i++ {
			if ops[i] != ops[i+1] {
				return false, nil
			}
		}
		return true, nil
	case Diff:
		if len(c.Op) <= 1 {
			return false, fmt.Errorf("you need at least two operands to check equality")
		}
		ops, err := c.findInputs(input)
		if err != nil {
			return false, err
		}
		for i := 0; i < len(ops)-1; i++ {
			if ops[i] == ops[i+1] {
				return false, nil
			}
		}
		return true, nil
	case Greater:
		if len(c.Op) != 2 {
			return false, fmt.Errorf("you need exactly two numbers to check which is greater")
		}
		ops, err := c.findInputs(input)
		if err != nil {
			return false, err
		}
		// try converting operands to float
		f := function.Float{}
		float1, err1 := f.Convert(ops[0])
		float2, err2 := f.Convert(ops[1])
		if err1 == nil && err2 == nil {
			return float1 > float2, nil
		}
		// try converting operands to string
		t := function.Text{}
		string1, err3 := t.Convert(ops[0])
		string2, err4 := t.Convert(ops[1])
		if err3 == nil && err4 == nil {
			if string1 == "" || string2 == "" {
				return false, nil
			}
			// golang check strings with lexicographic order with the > operator
			return string1 > string2, nil
		}
		if err3 != nil {
			fmt.Printf("condition Greater: first operand '%v' cannot be converted to string\n", c.Op[0])
			return false, nil
		} else {
			fmt.Printf("condition Greater: second operand '%v' cannot be converted to string\n", c.Op[1])
			return false, nil
		}
	case Smaller:
		if len(c.Op) != 2 {
			return false, fmt.Errorf("you need exactly two numbers to check which is greater")
		}
		ops, err := c.findInputs(input)
		if err != nil {
			return false, err
		}
		// try converting operands to float
		f := function.Float{}
		float1, err1 := f.Convert(ops[0])
		float2, err2 := f.Convert(ops[1])
		if err1 == nil && err2 == nil {
			return float1 < float2, nil
		}
		// try converting operands to string
		t := function.Text{}
		string1, err3 := t.Convert(ops[0])
		string2, err4 := t.Convert(ops[1])
		if err3 == nil && err4 == nil {
			// golang check strings with lexicographic order with the > operator
			if string1 == "" || string2 == "" {
				return false, nil
			}
			return string1 < string2, nil
		}
		if err3 != nil {
			fmt.Printf("condition Smaller: first operand '%v' cannot be converted to string\n", c.Op[0])
			return false, nil
		} else {
			fmt.Printf("condition Smaller: second operand '%v' cannot be converted to string\n", c.Op[1])
			return false, nil
		}
	case Empty:
		ops, err := c.findInputs(input)
		if err != nil {
			return false, err
		}
		slice, err := utils.ConvertToSlice(ops[0])
		if err != nil {
			return false, err
		}
		return len(slice) == 0, nil
	case IsNumeric:
		isNumeric := func(s interface{}) bool {
			_, err := function.Int{}.Convert(s)
			if err == nil {
				return true
			}
			numericStringMaybe, ok := s.(string)
			if !ok {
				return false
			}
			_, err = strconv.ParseFloat(numericStringMaybe, 64)
			return err == nil
		}

		ops, err := c.findInputs(input)
		if err != nil {
			return false, err
		}
		return isNumeric(ops[0]), nil
	case StringMatches:
		ops, err := c.findInputs(input)
		inputString, okString := ops[0].(string)
		if !okString {
			return false, fmt.Errorf("condition StringMatches: first operand (string to match) is not a string")
		}
		pattern, okPattern := ops[1].(string)
		if !okPattern {
			return false, fmt.Errorf("condition StringMatches: second operand (pattern) is not a string")
		}

		// Replace \\* with a placeholder to treat it as a literal *
		pattern = strings.ReplaceAll(pattern, "\\\\*", "\x00")

		// Replace \\\\ with a placeholder to treat it as a literal \
		pattern = strings.ReplaceAll(pattern, "\\\\\\\\", "\x01")

		// Check for unescaped single backslashes
		if strings.Contains(pattern, "\\") {
			return false, fmt.Errorf("runtime error: open escape sequence found")
		}

		// Escape special regex characters in the pattern except for "*"
		escapedPattern := regexp.QuoteMeta(pattern)

		// Replace placeholder \x00 back to literal * in the escaped pattern
		escapedPattern = strings.ReplaceAll(escapedPattern, "\x00", "\\*")

		// Replace placeholder \x01 back to literal \ in the escaped pattern
		escapedPattern = strings.ReplaceAll(escapedPattern, "\x01", "\\\\")

		// Replace "*" with ".*" to match zero or more characters
		regexPattern := strings.ReplaceAll(escapedPattern, "\\*", ".*")

		// Compile the regex pattern
		re, err := regexp.Compile("^" + regexPattern + "$")
		if err != nil {
			return false, err
		}

		// Check if the input matches the regex pattern
		return re.MatchString(inputString), nil
	default:
		return false, fmt.Errorf("invalid operation condition %s", c.Type)
	}
}

func (p Predicate) Equals(o Predicate) bool {
	return p.Root.Equals(o.Root)
}
func (c Condition) Equals(o Condition) bool {
	typeOk := c.Type == o.Type
	if !typeOk {
		fmt.Printf("operand type is not the same: %s vs %s\n", c.Type, o.Type)
	}
	lenOpOk := len(c.Op) == len(o.Op)
	if !lenOpOk {
		fmt.Printf("operand size is not the same: %d vs %d\n", len(c.Op), len(o.Op))
	}
	lenFindOk := len(c.Find) == len(o.Find)
	if len(c.Op) > 0 && len(o.Op) > 0 && lenOpOk {
		for i := range c.Op {
			f := function.Float{}

			val1, err1 := f.Convert(c.Op[i])
			val2, err2 := f.Convert(o.Op[i])

			// checking value of Op elements (converting to float because JSON treats all number as floats)
			if val1 != val2 {
				if err1 != nil {
					fmt.Printf("convertion error1: %v", err1)
				}
				if err2 != nil {
					fmt.Printf("convertion error2: %v", err2)
				}
				return false
			}
		}
	}

	lenSubCondOk := len(c.Sub) == len(o.Sub)
	if !lenSubCondOk {
		fmt.Printf("subconditions size is not the same: %d vs %d\n", len(c.Sub), len(o.Sub))
	}
	subOk := true
	if lenSubCondOk {
		for i := range c.Sub { // eq/const/ non ha sotto condizioni
			subOk = subOk && c.Sub[i].Equals(o.Sub[i])
		}
	}

	return typeOk && lenOpOk && lenSubCondOk && lenFindOk && subOk
}

func NewConstCondition(val interface{}) Condition {
	b := function.Bool{}

	err := b.TypeCheck(val)
	if err != nil {
		return NewConstCondition(false)
	}

	return Condition{
		Type: Const,
		Op:   []interface{}{val},
		Find: []bool{false},
	}
}

func NewAnd(conditions ...Condition) Condition {
	return Condition{
		Type: And,
		Sub:  conditions,
		Find: make([]bool, len(conditions)), // all false
	}
}

func NewOr(conditions ...Condition) Condition {
	return Condition{
		Type: Or,
		Sub:  conditions,
		Find: make([]bool, len(conditions)), // all false
	}
}

func NewNot(condition Condition) Condition {
	return Condition{
		Type: Not,
		Sub:  []Condition{condition},
		Find: make([]bool, 1), // all false
	}
}

// operations
func NewEqCondition(val1 interface{}, val2 interface{}) Condition {
	return Condition{
		Type: Eq,
		Op:   []interface{}{val1, val2},
		Find: make([]bool, 2), // all false
	}
}

type ParamOrValue struct {
	IsParam bool
	name    string
	value   interface{}
}

func NewParam(name string) *ParamOrValue {
	return &ParamOrValue{
		IsParam: true,
		name:    name,
		value:   nil,
	}
}

func NewValue(val interface{}) *ParamOrValue {
	return &ParamOrValue{
		IsParam: false,
		name:    "",
		value:   val,
	}
}

func (pv *ParamOrValue) GetOperand() interface{} {
	if pv.IsParam {
		return pv.name
	} else {
		return pv.value
	}
}

func NewEqParamCondition(param1 *ParamOrValue, param2 *ParamOrValue) Condition {
	ops := make([]interface{}, 0, 2)
	finds := make([]bool, 0, 2)
	params := []*ParamOrValue{param1, param2}
	for _, param := range params {
		finds = append(finds, param.IsParam)
		ops = append(ops, param.GetOperand())
	}
	return Condition{
		Type: Eq,
		Op:   ops,
		Find: finds,
	}
}

func NewDiffCondition(val1, val2 interface{}) Condition {
	return Condition{
		Type: Diff,
		Op:   []interface{}{val1, val2},
		Find: make([]bool, 2), // all false
	}
}

func NewDiffParamCondition(param1 *ParamOrValue, param2 *ParamOrValue) Condition {
	ops := make([]interface{}, 0, 2)
	finds := make([]bool, 0, 2)
	params := []*ParamOrValue{param1, param2}
	for _, param := range params {
		finds = append(finds, param.IsParam)
		ops = append(ops, param.GetOperand())
	}
	return Condition{
		Type: Diff,
		Op:   ops,
		Find: finds,
	}
}

func NewGreaterCondition(val1 interface{}, val2 interface{}) Condition {
	f := function.Float{}
	err1 := f.TypeCheck(val1)
	err2 := f.TypeCheck(val2)
	if err1 == nil && err2 == nil {
		return Condition{
			Type: Greater,
			Op:   []interface{}{val1, val2},
			Find: make([]bool, 2), // all false
		}
	}
	// let's try with strings
	b := function.Text{}
	err3 := b.TypeCheck(val1)
	err4 := b.TypeCheck(val2)
	if err3 == nil || err4 == nil {
		return Condition{
			Type: Greater,
			Op:   []interface{}{val1, val2},
			Find: make([]bool, 2), // all false
		}

	}
	fmt.Printf("cannot convert values neighter to float nor to string: %v, %v\n", val1, val2)
	return NewConstCondition(false)
}

func NewGreaterParamCondition(param1 *ParamOrValue, param2 *ParamOrValue) Condition {
	ops := make([]interface{}, 0, 2)
	finds := make([]bool, 0, 2)
	params := []*ParamOrValue{param1, param2}
	for _, param := range params {
		finds = append(finds, param.IsParam)
		ops = append(ops, param.GetOperand())
	}
	return Condition{
		Type: Greater,
		Op:   ops,
		Find: finds,
	}
}

func NewSmallerCondition(val1 interface{}, val2 interface{}) Condition {
	f := function.Float{}
	err1 := f.TypeCheck(val1)
	err2 := f.TypeCheck(val2)
	if err1 == nil && err2 == nil {
		return Condition{
			Type: Smaller,
			Op:   []interface{}{val1, val2},
			Find: make([]bool, 2), // all false
		}
	}
	// let's try with strings
	b := function.Text{}
	err3 := b.TypeCheck(val1)
	err4 := b.TypeCheck(val2)
	if err3 == nil || err4 == nil {
		return Condition{
			Type: Smaller,
			Op:   []interface{}{val1, val2},
			Find: make([]bool, 2), // all false
		}

	}
	fmt.Printf("cannot convert values neighter to float nor to string: %v, %v\n", val1, val2)
	return NewConstCondition(false)
}

func NewSmallerParamCondition(param1 *ParamOrValue, param2 *ParamOrValue) Condition {
	ops := make([]interface{}, 0, 2)
	finds := make([]bool, 0, 2)
	params := []*ParamOrValue{param1, param2}
	for _, param := range params {
		finds = append(finds, param.IsParam)
		ops = append(ops, param.GetOperand())
	}
	return Condition{
		Type: Smaller,
		Op:   ops,
		Find: finds,
	}
}

func NewEmptyCondition(collection []interface{}) Condition {
	return Condition{
		Type: Empty,
		Op:   collection,
		Find: make([]bool, 1), // all false,
	}
}

func NewIsNumericParamCondition(param1 *ParamOrValue) Condition {
	return Condition{
		Type: IsNumeric,
		Find: []bool{param1.IsParam},
		Op:   []interface{}{param1.GetOperand()},
	}
}

func NewStringMatchesParamCondition(param1 *ParamOrValue, param2 *ParamOrValue) Condition {
	return Condition{
		Type: StringMatches,
		Find: []bool{param1.IsParam, param2.IsParam},
		Op:   []interface{}{param1.GetOperand(), param2.GetOperand()},
	}
}

//func NewEmptyParamCondition(input map[string]interface{}, param1 string) Condition {
//	val1, found := input[param1]
//	if !found {
//		return NewConstCondition(false)
//	}
//	slice, errSlice := utils.ConvertToSlice(val1)
//	if errSlice != nil {
//		return NewConstCondition(false)
//	}
//	return NewEmptyCondition(slice)
//}

type ConditionBuilder struct {
	p      *Predicate
	errors string
}

type RootConditionBuilder struct {
	cb *ConditionBuilder
}

func NewPredicate() *RootConditionBuilder {
	return &RootConditionBuilder{cb: &ConditionBuilder{p: &Predicate{}}}
}

func (rcb *RootConditionBuilder) And(conds ...Condition) *ConditionBuilder {
	and := NewAnd(conds...)
	rcb.cb.p.Root = and
	return rcb.cb
}

func (rcb *RootConditionBuilder) Or(conds ...Condition) *ConditionBuilder {
	or := NewOr(conds...)
	rcb.cb.p.Root = or
	return rcb.cb
}

func (rcb *RootConditionBuilder) Not(cond Condition) *ConditionBuilder {
	not := NewNot(cond)
	rcb.cb.p.Root = not
	return rcb.cb
}

func (cb *ConditionBuilder) Build() Condition {
	return cb.p.Root
}
