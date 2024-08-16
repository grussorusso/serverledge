package asl

import (
	"fmt"
	"strings"
)

// Path is a string beginning with "$", used to identify components in a JSON text, in JSONPath format.
// Used to navigate an input parameter for a state
type Path string

// NewReferencePath creates a new Reference Path, that starts with a $ character, separated by "." characters and
// that does not contain the following characters '@' ',' ':' '?'. Used to define input or output parameters.
func NewReferencePath(s string) (Path, error) {
	if s == "" || s == "$" {
		return Path(s), nil
	}
	if !strings.HasPrefix(s, "$.") {
		s = "$." + s
	}
	if strings.Contains(s, "@") || strings.Contains(s, ",") || strings.Contains(s, ":") || strings.Contains(s, "?") {
		return "", fmt.Errorf("A reference path should not contain any of the following characters: '@' ',' ':' '?' ")
	}
	return Path(s), nil
}
