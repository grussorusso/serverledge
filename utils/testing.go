package utils

import (
	"testing"
)

func AssertEquals(t *testing.T, expected interface{}, result interface{}) {
	if expected != result {
		t.Logf("%s is failed. Got '%v', expected %v", t.Name(), result, expected)
		t.FailNow()
	}
}

func AssertNil(t *testing.T, result interface{}) {
	if nil != result {
		t.Logf("%s is failed. Got '%v', expected nil", t.Name(), result)
		t.FailNow()
	}
}

func AssertNonNil(t *testing.T, result interface{}) {
	if nil == result {
		t.Logf("%s is failed. Got '%v', expected non-nil", t.Name(), result)
		t.FailNow()
	}
}

// AssertNotEmptySlice asserts that a slice is not empty. Notice: here we use generics. Only works for go 1.19+
func AssertNotEmptySlice[A any](t *testing.T, slice []*A) {
	if slice == nil {
		t.Logf("%s is failed. The slice is nil,", t.Name())
		t.FailNow()
	}
	if len(slice) == 0 {
		t.Logf("%s is failed. The slice is empty,", t.Name())
		t.FailNow()
	}
}

func AssertEmptySlice(t *testing.T, slice []interface{}) {
	if slice == nil {
		t.Logf("%s is failed. The slice is nil,", t.Name())
		t.FailNow()
	}
	if len(slice) != 0 {
		t.Logf("%s is failed. The slice is NOT empty,", t.Name())
		t.FailNow()
	}
}

func AssertTrue(t *testing.T, isTrue bool) {
	if !isTrue {
		t.Logf("%s is failed. Got false", t.Name())
		t.FailNow()
	}
}

func AssertFalse(t *testing.T, isTrue bool) {
	if isTrue {
		t.Logf("%s is failed. Got false", t.Name())
		t.FailNow()
	}
}
