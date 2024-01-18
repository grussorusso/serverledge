package cli

import (
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
)

// ReadFromYAML TODO: we should read the yaml source, parse the contents and produce a Dag and a list of function
// returns the parsed dag, the list of distinct function if successful, or else an error
func ReadFromYAML(yamlSrc string) (*fc.Dag, []*function.Function, error) {
	// for now, we simply try different default dags
	return exampleParsing(yamlSrc)
}
