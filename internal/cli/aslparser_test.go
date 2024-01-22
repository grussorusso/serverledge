package cli_test

import (
	"fmt"
	"testing"

	"github.com/enginyoyen/aslparser"
)

func Test1(t *testing.T) {
	stateMachine, err := aslparser.ParseFile("../../test/machine.json", false)
	if err != nil {
		t.Fail()
		return
	}

	fmt.Printf("%v\n", stateMachine)
	if !stateMachine.Valid() {
		for _, e := range stateMachine.Errors() {
			fmt.Print(e.Description())
		}
	}
}
