package cli

import (
	"github.com/grussorusso/serverledge/internal/fc"
	"github.com/grussorusso/serverledge/internal/function"
)

// ReadFromJSON TODO: we should read the json source, parse the contents and produce a Dag and a list of function
// returns the parsed dag and the list of distinct function if successful, or else an error
func ReadFromJSON(jsonSrc string) (*fc.Dag, []*function.Function, error) {
	//stateMachine, err := aslparser.ParseFile(jsonSrc, false)
	//if err != nil {
	//	return nil, nil, fmt.Errorf("Could not parse the ASL file")
	//}
	//fmt.Println("Parsed ASL JSON")

	//if !stateMachine.Valid() {
	//	for _, e := range stateMachine.Errors() {
	//		fmt.Print(e.Description())
	//		return nil, nil, fmt.Errorf("Invalid ASL file")
	//	}
	//}

	//for _, s := range stateMachine.States {
	//	if s.Type == "Task" {

	//	} else {
	//		fmt.Printf("Unsupported task type: %s\n", s)
	//	}
	//}

	//builder := fc.NewDagBuilder()
	//for _, f := range funcs {
	//	builder = builder.AddSimpleNode(f)
	//}

	//dag, err := builder.Build()
	//return dag, nil, nil

	return exampleParsing(jsonSrc)
}
