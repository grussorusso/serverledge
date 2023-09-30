package cli

import (
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
)

// ReadFromJSON TODO: we should read the json source, parse the contents and produce a Dag and a list of function
// returns the parsed dag and the list of distinct function if successful, or else an error
func ReadFromJSON(jsonSrc string) (*fc.Dag, []*function.Function, error) {
	return exampleParsing(jsonSrc)
}
