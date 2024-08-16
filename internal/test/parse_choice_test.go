package test

import (
	"github.com/grussorusso/serverledge/internal/asl"
	"github.com/grussorusso/serverledge/utils"
	"testing"
)

func TestParseTerminalChoiceWithTwoBranches(t *testing.T) {
	t.Skip("WIP")
	choice := []byte(`{
		"Comment": "An example of the Amazon States Language using a choice state.",
		"StartAt": "FirstState",
		"States": {
			"FirstState": {
				"Comment": "Lang=Python",
				"Type": "Task",
				"Resource": "inc",
				"Next": "ChoiceState"
			},
			"ChoiceState": {
				"Type": "Choice",
				"Choices": [
					{
						"Variable": "input",
						"NumericEquals": 1,
						"Next": "FirstMatchState"
					},
					{
						"Variable": "input",
						"NumericEquals": 2,
						"Next": "SecondMatchState"
					}
				],
				"Default": "DefaultState"
			},
			"FirstMatchState": {
				"Comment": "Lang=Python",
				"Type": "Task",
				"Resource": "inc",
				"Next": "NextState"
			},	
			"SecondMatchState": {
				"Comment": "Lang=Python",
				"Type": "Task",
				"Resource": "double",
				"Next": "NextState"
			},
			
			"DefaultState": {
				"Type": "Succeed",
				"Cause": "No Matches!"
			},
			
			"NextState": {
				"Comment": "Lang=Python",
				"Type": "Task",
				"Resource": "hello",
				"End": true
			}
		}
	}`)

	sm, err := asl.ParseFrom("choice", choice)
	utils.AssertNilMsg(t, err, "failed to parse state machine")

	smExpected := &asl.StateMachine{
		StartAt: "FirstState",
		Comment: "A choice state machine with 1 task state",
		Version: "1.0",
		Name:    "choice",
		States: map[string]asl.State{
			"FirstState": asl.NewTerminalTask("inc"),
		},
	}
	utils.AssertTrueMsg(t, smExpected.Equals(sm), "state machines differs")
}
