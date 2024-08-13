package asl

type StateMachine struct {
	Comment string
	States  map[string]State
	StartAt string
	Version string
	// validationResult *gojsonschema.Result
}
