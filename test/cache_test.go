package test

import (
	"github.com/grussorusso/serverledge/internal/cache"
	"strconv"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	tc := cache.New(cache.DefaultExpiration, 1*time.Second, 2)

	tc.Set("a", 1, cache.DefaultExpiration)
	tc.Set("b", 2, cache.DefaultExpiration)
	tc.Set("c", 3, cache.DefaultExpiration)

	x, found := tc.Get("a")
	if !found {
		t.Log("a was selected to be replaced by using LRU policy\n")
	}

	x, found = tc.Get("c")
	if !found {
		t.Error("c was not found while getting c2")
	} else {
		c2 := x.(int)
		t.Log("c founded value: " + strconv.Itoa(c2))
	}

	x, found = tc.Get("b")
	if !found {
		t.Error("b was not found while getting b2")
	} else {
		b2 := x.(int)
		t.Log("b founded value: " + strconv.Itoa(b2))
	}

	tc.Set("a", 1, 5*time.Second)

	t.Log(tc.Items())
	t.Log("c was selected to be replaced by using LRU policy")

	time.Sleep(4 * time.Second)

	t.Log(tc.Items())

	x, found = tc.Get("b")
	if !found {
		t.Log("b was correctly cleaned by the janitor.")
	} else {
		t.Error("b not cleaned by janitor.")
	}

	x, found = tc.Get("a")
	if !found {
		t.Error("a not founded")
	} else {
		a2 := x.(int)
		t.Log("a founded value: " + strconv.Itoa(a2))

	}

	time.Sleep(2 * time.Second)
	t.Log(tc.Items())
	t.Log("a cleaned by the janitor")
}
