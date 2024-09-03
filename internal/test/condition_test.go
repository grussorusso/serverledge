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
var predicate4 = fc.Predicate{Root: fc.Condition{Type: fc.Not, Find: []bool{false}, Sub: []fc.Condition{{Type: fc.Empty, Op: []interface{}{1, 2, 3, 4}, Find: []bool{false}}}}}

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
	utils.AssertEquals(t, "!(empty input)", str4)
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
	isNumeric := fc.NewIsNumericParamCondition(fc.NewValue("123"))
	ok, err := isNumeric.Test(map[string]interface{}{})
	utils.AssertNil(t, err)
	utils.AssertTrue(t, ok)

	isNumeric = fc.NewIsNumericParamCondition(fc.NewParam("foo"))
	testMap := make(map[string]interface{})
	testMap["foo"] = "123"
	ok, err = isNumeric.Test(testMap)
	utils.AssertNil(t, err)
	utils.AssertTrue(t, ok)

	isNumeric = fc.NewIsNumericParamCondition(fc.NewParam("foo"))
	testMap = make(map[string]interface{})
	testMap["foo"] = "bar"
	ok, err = isNumeric.Test(testMap)
	utils.AssertNil(t, err)
	utils.AssertFalse(t, ok)

	isNumeric = fc.NewIsNumericParamCondition(fc.NewParam("foo"))
	testMap = make(map[string]interface{})
	ok, err = isNumeric.Test(testMap)
	// foo is not specified, so ok should be false. No errors should be expected
	utils.AssertNil(t, err)
	utils.AssertFalse(t, ok)
}

func TestStringGreaterAndSmaller(t *testing.T) {
	// apple is not greater than banana
	isGreater := fc.NewGreaterCondition("apple", "banana")
	ok, err := isGreater.Test(map[string]interface{}{})
	utils.AssertNil(t, err)
	utils.AssertFalse(t, ok)

	// banana is greater than apple
	isGreater = fc.NewGreaterCondition("banana", "apple")
	ok, err = isGreater.Test(map[string]interface{}{})
	utils.AssertNil(t, err)
	utils.AssertTrue(t, ok)

	// apple is smaller than banana
	isSmaller := fc.NewSmallerCondition("apple", "banana")
	ok, err = isSmaller.Test(map[string]interface{}{})
	utils.AssertNil(t, err)
	utils.AssertTrue(t, ok)

	// banana is not smaller than apple
	isSmaller = fc.NewSmallerCondition("banana", "apple")
	ok, err = isSmaller.Test(map[string]interface{}{})
	utils.AssertNil(t, err)
	utils.AssertFalse(t, ok)

	/* Corner cases */

	// banana is not greater than banana
	isGreater = fc.NewGreaterCondition("banana", "banana")
	ok, err = isGreater.Test(map[string]interface{}{})
	utils.AssertNil(t, err)
	utils.AssertFalse(t, ok)

	// banana is not greater than banana
	isSmaller = fc.NewSmallerCondition("banana", "banana")
	ok, err = isSmaller.Test(map[string]interface{}{})
	utils.AssertNil(t, err)
	utils.AssertFalse(t, ok)

	// nil is not smaller than banana
	isSmaller = fc.NewSmallerCondition(nil, "banana")
	ok, err = isSmaller.Test(map[string]interface{}{})
	utils.AssertNil(t, err)
	utils.AssertFalse(t, ok)

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
