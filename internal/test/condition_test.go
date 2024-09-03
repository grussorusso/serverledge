package test

import (
	"encoding/json"
	"fmt"
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/utils"
	"testing"
)

var predicate1 = fc.Predicate{Root: fc.Condition{Type: fc.And, Find: []bool{false, false}, Sub: []fc.Condition{{Type: fc.Eq, Op: []interface{}{2, 2}, Find: []bool{false, false}}, {Type: fc.Greater, Op: []interface{}{4, 2}, Find: []bool{false, false}}}}}
var predicate2 = fc.Predicate{Root: fc.Condition{Type: fc.Or, Find: []bool{false, false}, Sub: []fc.Condition{{Type: fc.Const, Op: []interface{}{true}, Find: []bool{false}}, {Type: fc.Smaller, Op: []interface{}{4, 2}, Find: []bool{false, false}}}}}
var predicate3 = fc.Predicate{Root: fc.Condition{Type: fc.Or, Find: []bool{false, false}, Sub: []fc.Condition{predicate1.Root, {Type: fc.Smaller, Op: []interface{}{4, 2}, Find: []bool{false, false}}}}}
var predicate4 = fc.Predicate{Root: fc.Condition{Type: fc.Not, Find: []bool{false}, Sub: []fc.Condition{{Type: fc.IsEmpty, Op: []interface{}{1, 2, 3, 4}, Find: []bool{false}}}}}

func TestPredicateMarshal(t *testing.T) {

	predicates := []fc.Predicate{predicate1, predicate2, predicate3, predicate4}
	for _, predicate := range predicates {
		val, err := json.Marshal(predicate)
		utils.AssertNil(t, err)

		var predicateTest fc.Predicate
		errUnmarshal := json.Unmarshal(val, &predicateTest)
		utils.AssertNil(t, errUnmarshal)
		fmt.Printf("predicateInput\t: %+v\n", predicate)
		fmt.Printf("predicateTest\t: %+v\n", predicateTest)
		utils.AssertTrue(t, predicate.Equals(predicateTest))
	}
}

func TestPredicate(t *testing.T) {
	ok := predicate1.Test(nil)
	utils.AssertTrue(t, ok)

	ok2 := predicate2.Test(nil)
	utils.AssertTrue(t, ok2)

	ok3 := predicate3.Test(nil)
	utils.AssertTrue(t, ok3)

	ok4 := predicate4.Test(nil)
	utils.AssertTrue(t, ok4)
}

func TestPrintPredicate(t *testing.T) {
	str := predicate1.LogicString()
	utils.AssertEquals(t, "(2 == 2 && 4 > 2)", str)
	predicate1.Print()
	str2 := predicate2.LogicString()
	utils.AssertEquals(t, "(true || 4 < 2)", str2)
	predicate2.Print()

	str3 := predicate3.LogicString()
	utils.AssertEquals(t, "((2 == 2 && 4 > 2) || 4 < 2)", str3)
	predicate3.Print()

	str4 := predicate4.LogicString()
	utils.AssertEquals(t, "!(IsEmpty(1))", str4)
	predicate4.Print()
}

func TestBuilder(t *testing.T) {
	built1 := fc.NewPredicate().And(
		fc.NewEqCondition(2, 2),
		fc.NewGreaterCondition(4, 2),
	).Build()

	utils.AssertTrue(t, built1.Equals(predicate1.Root))

	built2 := fc.NewPredicate().Or(
		fc.NewConstCondition(true),
		fc.NewSmallerCondition(4, 2),
	).Build()

	utils.AssertTrue(t, built2.Equals(predicate2.Root))

	built3 := fc.NewPredicate().Or(
		fc.NewAnd(
			fc.NewEqCondition(2, 2),
			fc.NewGreaterCondition(4, 2),
		),
		fc.NewSmallerCondition(4, 2),
	).Build()
	utils.AssertTrue(t, built3.Equals(predicate3.Root))

	built4 := fc.NewPredicate().Not(
		fc.NewEmptyCondition([]interface{}{1, 2, 3, 4}),
	).Build()

	utils.AssertTrue(t, built4.Equals(predicate4.Root))

}

func TestIsNumeric(t *testing.T) {

	tests := []struct {
		parameter       string
		shouldBeNumeric bool
	}{
		{"name", false},
		{"age", true},
		{"height", true},
		{"phone", false},
		{"isStudent", false},
		{"nonExistent", false},
	}
	testMap := make(map[string]interface{})
	testMap["name"] = "John"
	testMap["age"] = 33
	testMap["height"] = 173.3
	testMap["phone"] = "3348176718"
	testMap["isStudent"] = false

	for i, test := range tests {
		cond := fc.NewIsNumericParamCondition(fc.NewParam(test.parameter))
		ok, err := cond.Test(testMap)
		utils.AssertNil(t, err)
		utils.AssertEqualsMsg(t, test.shouldBeNumeric, ok, fmt.Sprintf("test %d: expected IsString(%v) to be %v", i+1, test.parameter, test.shouldBeNumeric))
	}
}

func TestStringGreaterAndSmaller(t *testing.T) {
	tests := []struct {
		firstString    interface{}
		secondString   string
		firstIsGreater bool
		firstIsSmaller bool
	}{
		{"apple", "banana", false, true},
		{"banana", "apple", true, false},
		{"banana", "banana", false, false},
		{nil, "apple", false, false},
	}

	for i, test := range tests {
		isGreater := fc.NewGreaterCondition(test.firstString, test.secondString)
		isSmaller := fc.NewSmallerCondition(test.firstString, test.secondString)
		ok, err := isGreater.Test(map[string]interface{}{})
		ok2, err2 := isSmaller.Test(map[string]interface{}{})
		utils.AssertNil(t, err)
		utils.AssertNil(t, err2)
		utils.AssertEqualsMsg(t, test.firstIsGreater, ok, fmt.Sprintf("test %d:  when comparing %v > %v", i+1, test.firstString, test.secondString))
		utils.AssertEqualsMsg(t, test.firstIsSmaller, ok2, fmt.Sprintf("test %d: when comparing %v < %v", i+1, test.firstString, test.secondString))
	}

}

func TestStringMatches(t *testing.T) {
	tests := []struct {
		input   string
		pattern string
		match   bool
	}{
		{"foo23.log", "foo*.log", true},
		{"zebra.log", "*.log", true},
		{"foobar.zebra", "foo*.*", true},
		{"test.log", "foo*.log", false},
		{"foo.log", "*.txt", false},
		{"foo.log", "*.txt", false},
		{"fo*o.log", "fo\\\\*o.log", true},
	}

	for _, test := range tests {
		cond := fc.NewStringMatchesParamCondition(fc.NewValue(test.input), fc.NewValue(test.pattern))
		ok, err := cond.Test(map[string]interface{}{})
		utils.AssertNil(t, err)
		utils.AssertEqualsMsg(t, ok, test.match, fmt.Sprintf("expected %s to match %s", test.input, test.pattern))
	}
}

func TestBooleanEquals(t *testing.T) {
	tests := []struct {
		firstBoolean  interface{}
		secondBoolean interface{}
		equals        bool
	}{
		{true, true, true},
		{false, false, true},
		{"true", "true", true},
		{"true", true, false},
		{false, "false", false},
		{"true", 1, false},
		{false, 0, false},
		{"maybe", false, false},
		{2, true, false},
	}

	for i, test := range tests {
		cond := fc.NewEqParamCondition(fc.NewValue(test.firstBoolean), fc.NewValue(test.secondBoolean))
		ok, err := cond.Test(map[string]interface{}{})
		utils.AssertNil(t, err)
		utils.AssertEqualsMsg(t, ok, test.equals, fmt.Sprintf("test %d: expected %v to match %v", i+1, test.firstBoolean, test.secondBoolean))
	}
}

func TestIsNullIsPresent(t *testing.T) {
	tests := []struct {
		value       interface{}
		shouldBeNil bool
	}{
		{nil, true},
		{"null", true},
		{"", false},
		{[]byte{}, false},
		{0, false},
		{false, false},
	}

	for i, test := range tests {
		cond := fc.NewIsNullParamCondition(fc.NewValue(test.value))
		ok, err := cond.Test(map[string]interface{}{})
		utils.AssertNil(t, err)
		utils.AssertEqualsMsg(t, test.shouldBeNil, ok, fmt.Sprintf("test %d: expected IsNull(%v) to be %v", i+1, test.value, test.shouldBeNil))

		cond2 := fc.NewIsPresentParamCondition(fc.NewValue(test.value))
		ok2, err2 := cond2.Test(map[string]interface{}{})
		utils.AssertNil(t, err2)
		utils.AssertEqualsMsg(t, !test.shouldBeNil, ok2, fmt.Sprintf("test %d: expected IsNull(%v) to be %v", i+1, test.value, !test.shouldBeNil))
	}

	cond := fc.NewIsNullParamCondition(fc.NewParam("non-existent"))
	ok, err := cond.Test(map[string]interface{}{})
	utils.AssertNil(t, err)
	utils.AssertTrue(t, ok)

	cond2 := fc.NewIsPresentParamCondition(fc.NewParam("non-existent"))
	ok2, err2 := cond2.Test(map[string]interface{}{})
	utils.AssertNil(t, err2)
	utils.AssertFalse(t, ok2)
}

func TestIsString(t *testing.T) {
	tests := []struct {
		parameter      string
		shouldBeString bool
	}{
		{"name", true},
		{"age", false},
		{"isStudent", false},
		{"nonExistent", false},
	}
	testMap := make(map[string]interface{})
	testMap["name"] = "John"
	testMap["age"] = 33
	testMap["isStudent"] = false

	for i, test := range tests {
		cond := fc.NewIsStringParamCondition(fc.NewParam(test.parameter))
		ok, err := cond.Test(testMap)
		utils.AssertNil(t, err)
		utils.AssertEqualsMsg(t, ok, test.shouldBeString, fmt.Sprintf("test %d: expected IsString(%v) to be %v", i+1, test.parameter, test.shouldBeString))
	}
}
