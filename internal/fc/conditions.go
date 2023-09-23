package fc

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/function"
)

type Predicate struct {
	Root Condition
}

type Condition struct {
	Type CondEnum      `json:"Type"`
	Op   []interface{} `json:"Op"`
	Sub  []Condition   `json:"Sub"`
}

type CondEnum int

const (
	And = iota
	Or
	Not
	Const
	Eq
	Diff
	Greater
	Smaller
	Empty // for collections
)

func (p Predicate) Test() bool {
	ok, err := p.Root.Test()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
	}
	return ok
}

func (p Predicate) LogicString() string {
	return p.Root.ToString()
}

func (c Condition) ToString() string {
	switch c.Type {
	case And:
		str := "("
		for i, condition := range c.Sub {
			str += condition.ToString()
			if i != len(c.Sub)-1 {
				str += " && "
			}
		}
		str += ")"
		return str
	case Or:
		str := "("
		for i, condition := range c.Sub {
			str += fmt.Sprintf("%s", condition.ToString())
			if i != len(c.Sub)-1 {
				str += " || "
			}
		}
		str += ")"
		return str
	case Not:
		return fmt.Sprintf("!(%s)", c.Sub[0].ToString())
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
	default:
		return ""
	}
}

func (p Predicate) Print() {
	fmt.Println(p.LogicString())
}

func (c Condition) Test() (bool, error) {
	switch c.Type {
	case And:
		and := true
		for _, condition := range c.Sub {
			ok, err := condition.Test()
			if err != nil {
				return false, err
			}
			and = and && ok
		}
		return and, nil
	case Or:
		or := false
		for _, condition := range c.Sub {
			ok, err := condition.Test()
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

		test, err := c.Sub[0].Test()
		return !test, err
	case Const:
		if len(c.Op) == 0 {
			return false, fmt.Errorf("you need at least one const operand")
		}
		dataType := function.Bool{}
		return dataType.Convert(c.Op[0])
	case Eq:
		if len(c.Op) <= 1 {
			return false, fmt.Errorf("you need at least two operands to check equality")
		}
		for i := 0; i < len(c.Op)-1; i++ {
			if c.Op[i] != c.Op[i+1] {
				return false, nil
			}
		}
		return true, nil
	case Diff:
		if len(c.Op) <= 1 {
			return false, fmt.Errorf("you need at least two operands to check equality")
		}
		for i := 0; i < len(c.Op)-1; i++ {
			if c.Op[i] == c.Op[i+1] {
				return false, nil
			}
		}
		return true, nil
	case Greater:
		if len(c.Op) != 2 {
			return false, fmt.Errorf("you need exactly two numbers to check which is greater")
		}
		f := function.Float{}
		float1, err1 := f.Convert(c.Op[0])
		float2, err2 := f.Convert(c.Op[1])
		if err1 != nil || err2 != nil {
			return false, fmt.Errorf("not all operands %v can be converted to float64", c.Op)
		}
		return float1 > float2, nil
	case Smaller:
		if len(c.Op) != 2 {
			return false, fmt.Errorf("you need exactly two numbers to check which is greater")
		}
		f := function.Float{}
		float1, err1 := f.Convert(c.Op[0])
		float2, err2 := f.Convert(c.Op[1])
		if err1 != nil || err2 != nil {
			return false, fmt.Errorf("not all operands %v can be converted to float64", c.Op)
		}
		return float1 < float2, nil
	case Empty:
		return len(c.Op) == 0, nil
	default:
		return false, fmt.Errorf("invalid operation condition")
	}
}

func (p Predicate) Equals(o Predicate) bool {
	return p.Root.Equals(o.Root)
}
func (c Condition) Equals(o Condition) bool {
	typeOk := c.Type == o.Type
	if !typeOk {
		fmt.Printf("operand type is not the same: %d vs %d\n", c.Type, o.Type)
	}
	lenOpOk := len(c.Op) == len(o.Op)
	if !lenOpOk {
		fmt.Printf("operand size is not the same: %d vs %d\n", len(c.Op), len(o.Op))
	}
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

	return typeOk && lenOpOk && lenSubCondOk && subOk
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
	}
}

func NewAnd(conditions ...Condition) Condition {
	return Condition{
		Type: And,
		Sub:  conditions,
	}
}

func NewOr(conditions ...Condition) Condition {
	return Condition{
		Type: Or,
		Sub:  conditions,
	}
}

func NewNot(condition Condition) Condition {
	return Condition{
		Type: Not,
		Sub:  []Condition{condition},
	}
}

// operations
func NewEqCondition(val1 interface{}, val2 interface{}) Condition {
	return Condition{
		Type: Eq,
		Op:   []interface{}{val1, val2},
	}
}

func NewDiffCondition(val1, val2 interface{}) Condition {
	return Condition{
		Type: Diff,
		Op:   []interface{}{val1, val2},
	}
}

func NewGreaterCondition(val1 interface{}, val2 interface{}) Condition {
	b := function.Float{}
	err := b.TypeCheck(val1)
	err2 := b.TypeCheck(val2)
	if err != nil || err2 != nil {
		fmt.Printf("cannot convert values to float: %v, %v\n", val1, val2)
		return NewConstCondition(false)
	}
	return Condition{
		Type: Greater,
		Op:   []interface{}{val1, val2},
	}
}
func NewSmallerCondition(val1 interface{}, val2 interface{}) Condition {
	b := function.Float{}
	err := b.TypeCheck(val1)
	err2 := b.TypeCheck(val2)
	if err != nil || err2 != nil {
		fmt.Printf("cannot convert values to float: %v, %v\n", val1, val2)
		return NewConstCondition(false)
	}
	return Condition{
		Type: Smaller,
		Op:   []interface{}{val1, val2},
	}
}

func NewEmptyCondition(collection []interface{}) Condition {
	return Condition{
		Type: Empty,
		Op:   collection,
	}
}

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
