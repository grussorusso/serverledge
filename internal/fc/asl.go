package fc

/*** Adapted from https://github.com/enginyoyen/aslparser ***/
import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/asl"
)

// FromASL parses a AWS State Language specification file and returns a Function Composition with the corresponding Serverledge Dag
// The name of the composition should not be the file name by default, to avoid problems when adding the same composition multiple times.
func FromASL(name string, aslSrc []byte) (*FunctionComposition, error) {
	stateMachine, err := asl.ParseFrom(name, aslSrc)
	if err != nil {
		return nil, fmt.Errorf("could not parse the ASL file: %v", err)
	}
	return FromStateMachine(stateMachine, true)
}
