package test

import (
	"fmt"
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
			  "Resource": "inc",
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

func TestParseTwoTask(t *testing.T) {

	simple := []byte(`{
	  	"Comment": "A simple state machine with 2 task state",
	  	"StartAt": "FirstState",
	  	"States": {
	    	"FirstState": {
			  	"Comment": "The first task",
			  	"Type": "Task",
			  	"Resource": "inc",
				"Next": "SecondState",
	    	},
			"SecondState": {
			  "Comment": "The second task",
			  "Type": "Task",
			  "Resource": "inc",
			  "End": true
	    	},
	  	}
	}`)

	sm, err := asl.ParseFrom("simple", simple)
	utils.AssertNilMsg(t, err, "failed to parse state machine")

	smExpected := &asl.StateMachine{
		StartAt: "FirstState",
		Comment: "A simple state machine with 2 task state",
		Version: "1.0",
		Name:    "simple",
		States: map[string]asl.State{
			"FirstState":  asl.NewNonTerminalTask("inc", "SecondState"),
			"SecondState": asl.NewTerminalTask("inc"),
		},
	}
	utils.AssertTrue(t, smExpected.Equals(sm))
}

func TestParseTwoTaskInverted(t *testing.T) {

	simple := []byte(`{
	  	"Comment": "A simple state machine with 2 task state",
	  	"StartAt": "FirstState",
	  	"States": {
			"SecondState": {
			  "Comment": "The second task",
			  "Type": "Task",
			  "Resource": "inc",
			  "End": true
	    	},
			"FirstState": {
			  	"Comment": "The first task",
			  	"Type": "Task",
			  	"Resource": "inc",
				"Next": "SecondState",
	    	},
	  	}
	}`)

	sm, err := asl.ParseFrom("simple", simple)
	utils.AssertNilMsg(t, err, "failed to parse state machine")

	smExpected := &asl.StateMachine{
		StartAt: "FirstState",
		Comment: "A simple state machine with 2 task state",
		Version: "1.0",
		Name:    "simple",
		States: map[string]asl.State{
			"FirstState":  asl.NewNonTerminalTask("inc", "SecondState"),
			"SecondState": asl.NewTerminalTask("inc"),
		},
	}
	utils.AssertTrue(t, smExpected.Equals(sm))
}

func TestParseTaskWithoutEnd(t *testing.T) {
	simple := []byte(`{
	  	"Comment": "A simple state machine with 2 task state",
	  	"StartAt": "FirstState",
	  	"States": {
			"FirstState": {
			  	"Comment": "The first task",
			  	"Type": "Task",
			  	"Resource": "inc",
	    	},
	  	}
	}`)

	_, err := asl.ParseFrom("simple", simple)
	utils.AssertNonNilMsg(t, err, "parsing should have failed")
	fmt.Printf("Expected error: %v\n", err)
}

func TestParseTaskWithInvalidPath(t *testing.T) {
	simple := []byte(`{
	  	"Comment": "A simple state machine with 2 task state",
	  	"StartAt": "FirstState",
	  	"States": {
			"FirstState": {
			  	"Comment": "The first task",
				"Resource": "inc"
			  	"Type": "Task",
			  	"TimeoutSeconds": 10,
				"HeartbeatSecondsPath": "$.@<3heart"
			    "End": true
	    	},
	  	}
	}`)

	_, err := asl.ParseFrom("simple", simple)
	utils.AssertNonNilMsg(t, err, "parsing should have failed")
	fmt.Printf("Expected error: %v\n", err)
}

func TestParseTaskWithInvalidHeartbeat(t *testing.T) {
	simple := []byte(`{
	  	"Comment": "A simple state machine with 2 task state",
	  	"StartAt": "FirstState",
	  	"States": {
			"FirstState": {
			  	"Comment": "The first task",
			  	"Type": "Task",
			  	"Resource": "inc",
				"TimeoutSeconds": 5,
				"HeartbeatSeconds": 10,
			    "End": true
	    	},
	  	}
	}`)

	_, err := asl.ParseFrom("simple", simple)
	utils.AssertNonNilMsg(t, err, "parsing should have failed")
	fmt.Printf("Expected error: %v\n", err)
}

func TestParseTaskWithBothPathAndHeartbeat(t *testing.T) {
	simple := []byte(`{
	  	"Comment": "A simple state machine with 2 task state",
	  	"StartAt": "FirstState",
	  	"States": {
			"FirstState": {
			  	"Comment": "The first task",
			  	"Type": "Task",
			  	"Resource": "inc",
				"HeartbeatSecondsPath": "$.inc",
				"HeartbeatSeconds": 10,
			    "End": true
	    	},
	  	}
	}`)

	_, err := asl.ParseFrom("simple", simple)
	utils.AssertNonNilMsg(t, err, "parsing should have failed")
	fmt.Printf("Expected error: %v\n", err)
}

func TestParseTaskWithOnlyHeartbeat(t *testing.T) {
	simple := []byte(`{
	  	"Comment": "A simple state machine with 2 task state",
	  	"StartAt": "FirstState",
	  	"States": {
			"FirstState": {
			  	"Comment": "The first task",
			  	"Type": "Task",
			  	"Resource": "inc",
				"HeartbeatSeconds": 10,
			    "End": true
	    	},
	  	}
	}`)

	_, err := asl.ParseFrom("simple", simple)
	utils.AssertNonNilMsg(t, err, "parsing should have failed")
	fmt.Printf("Expected error: %v\n", err)
}
