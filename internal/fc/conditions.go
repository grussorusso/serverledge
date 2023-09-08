package fc

// Condition is something that can be evaluated to true or false
type Condition interface {
	Test() bool
}

type ConstCondition struct {
	Value bool
}

func (cc ConstCondition) Test() bool {
	return cc.Value
}

func NewConstCondition(value bool) Condition {
	return ConstCondition{value}
}

// Eq test equality of elements
type Eq[T comparable] struct {
	Params []T
}

func (e Eq[T]) Test() bool {
	ok := true
	for i := 0; i < len(e.Params)-1; i++ {
		a := e.Params[i]
		b := e.Params[i+1]
		ok = ok && a == b
	}

	return ok
}

func NewEqCondition[T comparable](a T, b T, more ...T) Condition {
	p := make([]T, 2+len(more))
	p[0] = a
	p[1] = b
	for i, m := range more {
		p[i+2] = m
	}
	return Eq[T]{Params: p}
}

type Number interface {
	int | int8 | int16 | int32 | int64 | float32 | float64
}

// Greater test if bigger Number is greater than smaller Number
type Greater[T Number] struct {
	Bigger  T
	Smaller T
}

func (g Greater[T]) Test() bool {
	return g.Bigger > g.Smaller
}

func NewGreaterCondition[T Number](big T, small T) Condition {
	return Greater[T]{Bigger: big, Smaller: small}
}

// Smaller test if bigger Number is greater than smaller Number
type Smaller[T Number] struct {
	Smaller T
	Bigger  T
}

func (g Smaller[T]) Test() bool {
	return g.Smaller < g.Bigger
}

func NewSmallerCondition[T Number](small T, big T) Condition {
	return Smaller[T]{Smaller: small, Bigger: big}
}

type And struct {
	Conditions []Condition
}

func NewAndCondition(condition1 Condition, condition2 Condition, conditions ...Condition) And {
	conditionList := make([]Condition, 2+len(conditions))
	conditionList[0] = condition1
	conditionList[1] = condition2
	for i, m := range conditions {
		conditionList[i+2] = m
	}
	return And{Conditions: conditionList}
}

func (a And) Test() bool {
	for _, c := range a.Conditions {
		if !c.Test() {
			return false
		}
	}
	return true
}

type Or[C Condition] struct {
	Conditions []C
}

func NewOrCondition[C Condition](condition1 C, condition2 C, conditions ...C) Or[C] {
	conditionList := make([]C, 2+len(conditions))
	conditionList[0] = condition1
	conditionList[1] = condition2
	for i, m := range conditions {
		conditionList[i+2] = m
	}
	return Or[C]{Conditions: conditionList}
}

func (a Or[C]) Test() bool {
	for _, c := range a.Conditions {
		if c.Test() {
			return true
		}
	}
	return false
}

type Not[C Condition] struct {
	Condition C
}

func NewNotCondition[C Condition](c C) Not[C] {
	return Not[C]{Condition: c}
}

func (n Not[C]) Test() bool {
	return !n.Condition.Test()
}
