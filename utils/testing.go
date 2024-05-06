package utils

import (
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"testing"
)

// AssertEquals verifies that the expected generic object T is equal to result T.
// If expected differs from result in any way, the test will fail immediately.
// The type of expected and result should be the same, and it should implement the comparable interface
// All Assert functions defined in this project work for go 1.19+ because they use generic types.
func AssertEquals[T comparable](t *testing.T, expected T, result T) {
	if expected != result {
		t.Logf("%s is failed. Got '%v', expected '%v'", t.Name(), result, expected)
		t.FailNow()
	}
}

// AssertEqualsMsg is like AssertEquals, but it also prints a custom message when the test fails.
func AssertEqualsMsg[T comparable](t *testing.T, expected T, result T, msg string) {
	if expected != result {
		t.Logf("%s is failed; %s - Got '%v', expected '%v'", t.Name(), msg, result, expected)
		t.FailNow()
	}
}

// AssertSliceEquals is like AssertEquals but works for slices
// Each element of the expected slice must be equal to the corresponding element in the result slice, in the same order.
func AssertSliceEquals[T comparable](t *testing.T, expected []T, result []T) {
	if equal := slices.Equal(expected, result); !equal {
		t.Logf("%s is failed Got '%v', expected '%v'", t.Name(), result, expected)
		t.FailNow()
	}
}

// AssertSliceEqualsMsg is like AssertSliceEquals, but it also prints a custom message when the test fails.
func AssertSliceEqualsMsg[T comparable](t *testing.T, expected []T, result []T, msg string) {
	if equal := slices.Equal(expected, result); !equal {
		t.Logf("%s is failed; %s - Got '%v', expected '%v'", t.Name(), msg, result, expected)
		t.FailNow()
	}
}

// AssertMapEquals is like AssertEqualsMsg but works for maps with key of type K, and value of type V.
// Both types must implement comparable. Every map should contain the same key-value pairs to make the test succeed.
func AssertMapEquals[K comparable, V comparable](t *testing.T, expectedMap map[K]V, resultMap map[K]interface{}) {
	typedMap := make(map[K]V)
	for k, v := range resultMap {
		typedMap[k] = v.(V)
	}
	if equal := maps.Equal(expectedMap, typedMap); !equal {
		t.Logf("%s is failed. Got '%v', expected '%v'", t.Name(), resultMap, expectedMap)
		t.FailNow()
	}
}

// AssertNil checks that result is nil. Useful for checking that there are no errors.
// If there is an error, it will fail the test immediately. It can also be used to expect nothing from a function
// (but you should never return nil from a function unless you like SIGSEGV!)
func AssertNil(t *testing.T, result interface{}) {
	if nil != result {
		t.Logf("%s is failed. Got '%v', expected nil", t.Name(), result)
		t.FailNow()
	}
}

// AssertNilMsg is like AssertNil, but it also prints a custom message when the test fails.
func AssertNilMsg(t *testing.T, result interface{}, msg string) {
	if nil != result {
		t.Logf("%s is failed; %s - Got '%v', expected nil", t.Name(), result, msg)
		t.FailNow()
	}
}

// AssertNonNil checks that result is non-nil. Useful for checking that there is some result,
// but we are not interested in its details.
func AssertNonNil(t *testing.T, result interface{}) {
	if nil == result {
		t.Logf("%s is failed. Got '%v', expected non-nil", t.Name(), result)
		t.FailNow()
	}
}

// AssertNonNilMsg is like AssertNonNil, but it also prints a custom message when the test fails.
func AssertNonNilMsg(t *testing.T, result interface{}, msg string) {
	if nil == result {
		t.Logf("%s is failed; %s - Got '%v', expected non-nil", t.Name(), result, msg)
		t.FailNow()
	}
}

// AssertNotEmptySlice asserts that a slice is non-nil and not empty, otherwise fails the test immediately.
func AssertNotEmptySlice[A any](t *testing.T, slice []A) {
	if slice == nil {
		t.Logf("%s is failed. The slice is nil,", t.Name())
		t.FailNow()
	}
	if len(slice) == 0 {
		t.Logf("%s is failed. The slice is empty,", t.Name())
		t.FailNow()
	}
}

// AssertEmptySlice asserts that a slice is empty, otherwise fails the test immediately.
func AssertEmptySlice[T any](t *testing.T, slice []T) {
	if slice == nil {
		t.Logf("%s is failed. The slice is nil,", t.Name())
		t.FailNow()
	}
	if len(slice) != 0 {
		t.Logf("%s is failed. The slice is NOT empty,", t.Name())
		t.FailNow()
	}
}

// AssertTrue verifies that given boolean is true, otherwise fails the test immediately
func AssertTrue(t *testing.T, isTrue bool) {
	if !isTrue {
		t.Logf("%s is failed. Got false", t.Name())
		t.FailNow()
	}
}

// AssertTrueMsg verifies that given boolean is true, otherwise fails the test immediately and prints a custom message
func AssertTrueMsg(t *testing.T, isTrue bool, msg string) {
	if !isTrue {
		t.Logf("%s is false - %s", t.Name(), msg)
		t.FailNow()
	}
}

// AssertFalse verifies that given boolean is false, otherwise fails the test immediately
func AssertFalse(t *testing.T, isTrue bool) {
	if isTrue {
		t.Logf("%s is failed. Got true", t.Name())
		t.FailNow()
	}
}

// AssertFalseMsg verifies that given boolean is false, otherwise fails the test immediately and prints a custom message
func AssertFalseMsg(t *testing.T, isTrue bool, msg string) {
	if isTrue {
		t.Logf("%s is true - %s", t.Name(), msg)
		t.FailNow()
	}
}
