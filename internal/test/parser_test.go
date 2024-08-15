package test

import (
	"github.com/grussorusso/serverledge/internal/asl"
	"github.com/grussorusso/serverledge/utils"
	"testing"
)

func TestParseSingleTerminalTask(t *testing.T) {

	simple := []byte(`{
	  	"Comment": "A simple state machine with 1 task state",
	  	"StartAt": "FirstState",
	  	"States": {
	    	"FirstState": {
			  "Comment": "The first task",
			  "Type": "Task",
			  "Resource": "$.inc",
			  "End": true
	    	},
	  	}
	}`)

	sm, err := asl.ParseFrom("simple", simple)
	utils.AssertNilMsg(t, err, "failed to parse state machine")

	smExpected := &asl.StateMachine{
		StartAt: "FirstState",
		Comment: "A simple state machine with 1 task state",
		Version: "1.0",
		Name:    "simple",
		States: map[string]asl.State{
			"FirstState": asl.NewTerminalTask("inc"),
		},
	}
	utils.AssertTrue(t, smExpected.Equals(sm))
}
