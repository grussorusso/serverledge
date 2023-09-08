package test

import (
	"encoding/json"
	"fmt"
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/utils"
	"testing"
)

func TestMarshalCondition(t *testing.T) {
	c := fc.NewAndCondition(fc.NewConstCondition(true), fc.NewEqCondition(2, 1), fc.NewSmallerCondition(1, 5))
	m, err := json.Marshal(c)
	utils.AssertNil(t, err)
	fmt.Printf("%s\n", m)
	utils.AssertEquals(t, "{\"Conditions\":[{\"Value\":true},{\"Params\":[2,1]},{\"Smaller\":1,\"Bigger\":5}]}", fmt.Sprintf("%s", m))

	//var and fc.And
	//err2 := json.Unmarshal(m, &and)
	//utils.AssertNil(t, err2)
	//
	//utils.AssertEquals(t, c, and)
	//fmt.Printf("%+v\n", and)
}

func TestConstCondition(t *testing.T) {
	c := fc.NewConstCondition(true)
	utils.AssertTrue(t, c.Test())

	c = fc.NewConstCondition(false)
	utils.AssertFalse(t, c.Test())
}

func TestEqCondition(t *testing.T) {
	c := fc.NewEqCondition(1, 1)
	utils.AssertTrue(t, c.Test())

	c = fc.NewEqCondition(1, 1, 1)
	utils.AssertTrue(t, c.Test())

	c = fc.NewEqCondition(1, 2, 1, 4)
	utils.AssertFalse(t, c.Test())
}

func TestGreaterCondition(t *testing.T) {
	c := fc.NewGreaterCondition(5, 4) // 5 > 4
	utils.AssertTrue(t, c.Test())

	c = fc.NewGreaterCondition(4, 4) // 4 > 4
	utils.AssertFalse(t, c.Test())

	c = fc.NewGreaterCondition(4.9, 4.5) // 4.9 > 4.5
	utils.AssertTrue(t, c.Test())

}

func TestSmallerCondition(t *testing.T) {
	c := fc.NewSmallerCondition(4, 5)
	utils.AssertTrue(t, c.Test())

	c = fc.NewSmallerCondition(4, 4)
	utils.AssertFalse(t, c.Test())

	c = fc.NewSmallerCondition(4.2, 4.5)
	utils.AssertTrue(t, c.Test())

}

func TestAndCondition(t *testing.T) {
	c := fc.NewAndCondition(fc.NewConstCondition(true), fc.NewEqCondition(1, 1))
	utils.AssertTrue(t, c.Test())

	c = fc.NewAndCondition(fc.NewConstCondition(true), fc.NewEqCondition(1, 1), fc.NewSmallerCondition(1, 5))
	utils.AssertTrue(t, c.Test())

	c = fc.NewAndCondition(fc.NewConstCondition(true), fc.NewEqCondition(2, 1), fc.NewSmallerCondition(1, 5))
	utils.AssertFalse(t, c.Test())

}

func TestOrCondition(t *testing.T) {
	c := fc.NewOrCondition(fc.NewConstCondition(true), fc.NewEqCondition(1, 1))
	utils.AssertTrue(t, c.Test())

	c = fc.NewOrCondition(fc.NewConstCondition(false), fc.NewEqCondition(1, 1), fc.NewSmallerCondition(1, 5))
	utils.AssertTrue(t, c.Test())

	c = fc.NewOrCondition(fc.NewConstCondition(false), fc.NewEqCondition(2, 1), fc.NewSmallerCondition(10, 5))
	utils.AssertFalse(t, c.Test())

}

func TestNotCondition(t *testing.T) {
	c := fc.NewNotCondition(fc.NewConstCondition(false))
	utils.AssertTrue(t, c.Test())

	c2 := fc.NewNotCondition(fc.NewAndCondition(fc.NewConstCondition(true), fc.NewEqCondition(2, 1)))
	utils.AssertTrue(t, c2.Test())

	c3 := fc.NewNotCondition(fc.NewEqCondition(2, 2))
	utils.AssertFalse(t, c3.Test())
}
