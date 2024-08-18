package test

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/asl"
	"github.com/grussorusso/serverledge/utils"
	"testing"
)

// TestParseTerminalChoiceWithThreeBranches tests that the following dag...
//
//	   [ Task ]
//	      |
//	   [Choice]
//	      |
//	(=1)--(=2)-(default)
//	 |     |	   |
//
// [Task][Task] [Task]
// ... is correctly parsed.
func TestParseChoiceWithThreeBranches(t *testing.T) {
	t.Skip("WIP")
	choice := []byte(`{
		"Comment": "An example of the Amazon States Language using a choice state.",
		"StartAt": "ChoiceState",
		"States": {
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
				"Comment": "Lang=Javascript",
				"Type": "Task",
				"Resource": "hello",
				"Next": "NextState"
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
		Comment: "An example of the Amazon States Language using a choice state.",
		Version: "1.0",
		Name:    "choice",
		States: map[string]asl.State{
			"ChoiceState": &asl.ChoiceState{
				Type: asl.Choice,
				Choices: []asl.ChoiceRule{
					&asl.DataTestExpression{
						Variable: "input",
						ComparisonOperator: &asl.ComparisonOperator{
							Kind:     asl.NumericEquals,
							DataType: asl.NumericComparator,
							Operand:  1,
						},
						Next: "FirstMatchState",
					},
					&asl.DataTestExpression{
						Variable: "input",
						ComparisonOperator: &asl.ComparisonOperator{
							Kind:     asl.NumericEquals,
							DataType: asl.NumericComparator,
							Operand:  2,
						},
						Next: "SecondMatchState",
					},
				},
				InputPath:  "",
				OutputPath: "",
				Default:    "DefaultSTate",
			},
			"FirstMatchState":  asl.NewNonTerminalTask("inc", "NextState"),
			"SecondMatchState": asl.NewNonTerminalTask("double", "NextState"),
			"DefaultState":     asl.NewNonTerminalTask("hello", "NextState"),
			"NextState":        asl.NewTerminalTask("hello"),
		},
	}
	ok := smExpected.Equals(sm)
	if !ok {
		fmt.Println("smExpected: ", smExpected)
		fmt.Println("smActual: ", sm)
	}
	utils.AssertTrueMsg(t, ok, "state machines differs")
}
