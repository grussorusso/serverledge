/*** Based on https://github.com/enginyoyen/aslparser ***/
package fc

/*
import (
	"encoding/json"
	"github.com/xeipuuv/gojsonschema"
	"io/ioutil"
)

// Given the file path validates and returns the StateMachine
// strict argument defines whether Resource name must be AWS ARN pattern or not
func ParseFile(filepath string, strict bool) (*StateMachine, error) {
	//load file
	payload, fileErr := ioutil.ReadFile(filepath)
	if fileErr != nil {
		return nil, fileErr
	}
	return Parse(payload, strict)
}

// Given the file content validates and returns the StateMachine
// strict argument defines whether Resource name must be AWS ARN pattern or not
func Parse(content []byte, strict bool) (*StateMachine, error) {

	// validate it, if there is an error or document is not Valid
	// return the result without further analysis
	var stateMachine StateMachine
	validationResult, valErr := Validate(content, strict)
	stateMachine.validationResult = validationResult
	if valErr != nil || !validationResult.Valid() {
		return &stateMachine, valErr
	}

	// given state-machine payload is valid, unmarshal the json file
	unmarshalErr := json.Unmarshal(content, &stateMachine)
	if unmarshalErr != nil {
		return &stateMachine, unmarshalErr
	}
	// find and register non-semantic errors
	stateMachine.findAndRegisterSchemaErrors()
	return &stateMachine, nil
}

// Valid returns true if no errors were found
func (s *StateMachine) Valid() bool {
	return s.validationResult.Valid()
}

// Errors returns the errors that were found
func (s *StateMachine) Errors() []gojsonschema.ResultError {
	return s.validationResult.Errors()
}

func (s *StateMachine) findAndRegisterSchemaErrors() {
	missingStates := *s.findNonSchemaErrors()

	for k, v := range missingStates {
		err := new(gojsonschema.MissingDependencyError)
		err.SetType("missing_dependency")
		err.SetDescriptionFormat(gojsonschema.Locale.MissingDependency())
		details := gojsonschema.ErrorDetails{
			"dependency": k + ". " + v,
		}
		err.SetDetails(details)
		s.validationResult.AddError(err, details)
	}
}

func (s *StateMachine) findNonSchemaErrors() *map[string]string {
	var errors = make(map[string]string)

	if !s.statePresent(s.StartAt) {
		errors[s.StartAt] = "Missing 'StartAt' transition target. Could not locate " + s.StartAt
	}

	for k, v := range s.States {
		if !s.targetStateRegistered(k) {
			errors[k] = k + " is defined but is not reachable."
		}
		if len(v.Next) > 0 && !s.statePresent(v.Next) {
			errors[k] = v.Next + " as Next,defined in " + k + ", but not reachable"
		}
		if len(v.Default) > 0 && !s.statePresent(v.Default) {
			errors[k] = v.Default + " as Default, defined in " + k + ", but not reachable"
		}
	}
	return &errors
}

func (s *StateMachine) statePresent(state string) bool {
	_, present := s.States[state]
	return present
}
func (s *StateMachine) targetStateRegistered(state string) bool {
	if s.StartAt == state {
		return true
	}

	match := false
	for _, v := range s.States {
		if len(v.Next) > 0 && v.Next == state {
			match = true
			break
		}

		if len(v.Default) > 0 && v.Next == state {
			match = true
			break
		}
	}
	return match
}
*/
