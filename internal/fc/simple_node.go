package fc

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/client"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
	"github.com/grussorusso/serverledge/internal/scheduling"
	"github.com/grussorusso/serverledge/internal/types"
	"github.com/lithammer/shortuuid"
	"sync"
	"time"
)

var compositionRequestsPool = sync.Pool{
	New: func() any {
		return new(function.Request)
	},
}

// SimpleNode is a DagNode that receives one input and sends one result
type SimpleNode struct {
	Id    string
	input map[string]interface{}
	// InputFrom DagNode // TODO: maybe we do not need it because from this we shouldn't be able to go back. We are creating an DIRECTED Acyclic Graph!
	OutputTo DagNode
	Func     string // *function.Function
	// Request   *function.Request
	// outputMappingPolicy OutMapPolicy  // this policy should be needed to decide how to map outputs to the next node
}

func NewSimpleNode(f string) *SimpleNode {
	return &SimpleNode{
		Id:   shortuuid.New(),
		Func: f,
		// Request: nil,
	}
}

func (s *SimpleNode) Equals(cmp types.Comparable) bool {
	switch cmp.(type) {
	case *SimpleNode:
		s2 := cmp.(*SimpleNode)
		idOk := s.Id == s2.Id
		// inputOk := s.InputFrom == s2.InputFrom
		funcOk := s.Func == s2.Func
		outputOk := s.OutputTo == s2.OutputTo
		return idOk && funcOk && outputOk // && inputOk
	default:
		return false
	}
}

func (s *SimpleNode) Exec() (map[string]interface{}, error) {
	funct, ok := function.GetFunction(s.Func)
	if !ok {
		return nil, fmt.Errorf("SimpleNode.function is null: you must initialize SimpleNode's function to execute it")
	}

	// creates the function if not exists. Maybe someone deleted by accident the function before starting the dag.
	//if !s.Func.Exists() {
	//	errNotSaved := s.Func.SaveToEtcd()
	//	return nil, fmt.Errorf("the function %s cannot be saved while trying to exec the function composition %v", s.Func.Name, errNotSaved)
	//}
	// the rest of the code is similar to a single function execution
	invocationRequest := client.InvocationRequest{
		Params:          s.input,
		QoSMaxRespT:     250,
		CanDoOffloading: true,
		Async:           false, // should always be synchronous. If we need to call the dag asynchronously, we can do that regardless.
		// TODO: aggiungere un campo che distingue se l'invocazione fa parte di una composizione o è una funzione singola
	}
	r := compositionRequestsPool.Get().(*function.Request) // function.Request will be created if does not exists, otherwise removed from the pool
	defer compositionRequestsPool.Put(r)                   // at the end of the function, the function.Request is added to the pool.

	// s.Request = r
	r.Fun = funct
	r.Params = invocationRequest.Params
	r.Arrival = time.Now()

	r.MaxRespT = invocationRequest.QoSMaxRespT
	r.CanDoOffloading = invocationRequest.CanDoOffloading
	r.Async = invocationRequest.Async
	r.ReqId = fmt.Sprintf("%s-%s%d", s.Func, node.NodeIdentifier[len(node.NodeIdentifier)-5:], r.Arrival.Nanosecond())
	// init fields if possibly not overwritten later
	r.ExecReport.SchedAction = ""
	r.ExecReport.OffloadLatency = 0.0

	// executes the function, waiting for the result
	err := scheduling.SubmitRequest(r)
	if err != nil {
		return nil, err
	}

	m := make(map[string]interface{})
	// extract output map
	for _, o := range funct.Signature.GetOutputs() {
		// if the output is a simple type (e.g. int, bool, string, array) we simply add it to the map
		m[o.Name] = r.ExecReport.Result
		err1 := o.CheckOutput(m)
		if err1 != nil {
			return nil, fmt.Errorf("output type checking failed: %v", err1)
		}
		m[o.Name], err1 = o.TryParse(r.ExecReport.Result)
		if err1 != nil {
			return nil, fmt.Errorf("failed to parse intermediate output: %v", err1)
		}
		// TODO: else if the output is a struct/map, we should return a map with struct field and values
	}
	cs := ""
	if !r.ExecReport.IsWarmStart {
		cs = fmt.Sprintf("- cold start: %v", !r.ExecReport.IsWarmStart)
	}
	fmt.Printf("result: %v %s\n", r.ExecReport.Result, cs)
	return m, nil
}

// AddInput connects a DagNode to the input of this SimpleNode TODO: REMOVE
func (s *SimpleNode) AddInput(dagNode DagNode) error {
	// s.InputFrom = dagNode
	return nil
}

// AddOutput connects the output of the SimpleNode to another DagNode
func (s *SimpleNode) AddOutput(dagNode DagNode) error {
	s.OutputTo = dagNode
	return nil
}

func (s *SimpleNode) ReceiveInput(input map[string]interface{}) error {
	// TODO: must communicate and receive input from other node, if on another machine
	funct, exists := function.GetFunction(s.Func) // we are getting the function from cache if not already downloaded
	if !exists {
		return fmt.Errorf("funtion %s doesn't exists", s.Func)
	}

	if funct.Signature == nil {
		return nil // signature is optional, but if set, you can debug errors more easily
	}

	err := funct.Signature.CheckAllInputs(input)
	if err != nil {
		return fmt.Errorf("error while receiving input: %v", err)
	}
	s.input = input
	return nil
}

func (s *SimpleNode) PrepareOutput(output map[string]interface{}) error {
	funct, exists := function.GetFunction(s.Func) // we are getting the function from cache if not already downloaded
	if !exists {
		return fmt.Errorf("funtion %s doesn't exists", s.Func)
	}
	err := funct.Signature.CheckAllOutputs(output)
	if err != nil {
		return fmt.Errorf("error while checking outputs: %v", err)
	}
	// get signature of next nodes, if present and maps the output there
	for _, n := range s.GetNext() {
		// TODO: this mapping should only be done with SimpleNode(s)? Yes, but this method must be implemented for all nodes
		// we have only one output node
		switch n.(type) {
		case *SimpleNode:
			return n.(*SimpleNode).MapOutput(output)
			//sign := n.(*SimpleNode).function.Signature
			//// if there are no inputs, we do nothing
			//for _, def := range sign.GetInputs() {
			//	// if output has same name as input, we do not need to change name
			//	_, present := output[def.Name]
			//	if present {
			//		continue
			//	}
			//	// find an entry in the output map that successfully checks the type of the InputDefinition
			//	key, ok := def.FindEntryThatTypeChecks(output)
			//	if ok {
			//		// we get the output value
			//		val := output[key]
			//		// we remove the output entry ...
			//		delete(output, key)
			//		// and replace with the input entry
			//		output[def.Name] = val
			//	} else {
			//		// otherwise if no one of the entry typechecks we are doomed
			//		return fmt.Errorf("no output entry input-checks with the next function")
			//	}
			//}
			//// if the outputs are more than the needed input, we do nothing
		}
	}

	return nil
}

func (s *SimpleNode) MapOutput(output map[string]interface{}) error {
	funct, exists := function.GetFunction(s.Func)
	if !exists {
		return fmt.Errorf("function %s doesn't exist", s.Func)
	}
	sign := funct.Signature
	// if there are no inputs, we do nothing
	for _, def := range sign.GetInputs() {
		// if output has same name as input, we do not need to change name
		_, present := output[def.Name]
		if present {
			continue
		}
		// find an entry in the output map that successfully checks the type of the InputDefinition
		key, ok := def.FindEntryThatTypeChecks(output)
		if ok {
			// we get the output value
			val := output[key]
			// we remove the output entry ...
			delete(output, key)
			// and replace with the input entry
			output[def.Name] = val
			// save the output map in the input of the node
			s.input = output
		} else {
			// otherwise if no one of the entry typechecks we are doomed
			return fmt.Errorf("no output entry input-checks with the next function")
		}
	}
	// if the outputs are more than the needed input, we do nothing
	return nil
}

func (s *SimpleNode) GetNext() []DagNode {
	// we only have one output
	arr := make([]DagNode, 1)
	if s.OutputTo != nil {
		arr[0] = s.OutputTo
		return arr
	}
	panic("you forgot to initialize next for StartNode")
}

func (s *SimpleNode) Width() int {
	return 1
}
func (s *SimpleNode) Name() string {
	return "Simple"
}

func (s *SimpleNode) ToString() string {
	return fmt.Sprintf("[SimpleNode (%s) func %s(%v)]->%s", s.Id, s.Func, s.input, s.OutputTo.Name())
}