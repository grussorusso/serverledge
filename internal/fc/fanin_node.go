package fc

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/types"
	"github.com/lithammer/shortuuid"
	"time"
)

type MergeMode int

const (
	AddNewMapEntry  = iota // The output type will be a map of key-values
	AddToArrayEntry        // The output type will be a map with a single array of values, with repetition
	AddToSetEntry          // The output type will be a map with a single array of unique values
)

// FanInNode receives and merges multiple input and produces a single result
type FanInNode struct {
	Id          DagNodeId
	NodeType    DagNodeType
	BranchId    int
	OutputTo    DagNodeId
	FanInDegree int
	Timeout     time.Duration
	Mode        MergeMode
	input       []map[string]interface{}
	//IsReached   bool
}

// FanInChannels is needed because we cannot marshal channels, so we need a different struct, that will be created each time a FanIn is used.
type FanInChannels struct {
	// Channels: used by simple nodes to send data to a fan in node
	Channels map[int]chan map[DagNodeId]interface{} // we need this double map because fan in should know which node to wait.
	// OutputChannel: used by fan in node to send merged output
	OutputChannel chan map[string]interface{}
}

// usedChannel is used by fanIn nodes
var usedChannels = make(map[DagNodeId]FanInChannels)

func createChannels(fanInId DagNodeId, fanInDegree int, branchNumbers []int) {
	// initializing the channel with branch numbers
	channels := make(map[int]chan map[DagNodeId]interface{})
	for i := 0; i < fanInDegree; i++ {
		channels[branchNumbers[i]] = make(chan map[DagNodeId]interface{})
	}
	usedChannels[fanInId] = FanInChannels{
		Channels:      channels,
		OutputChannel: make(chan map[string]interface{}),
	}
}

func getChannelForParallelBranch(fanInId DagNodeId, branchId int) chan map[DagNodeId]interface{} {
	return usedChannels[fanInId].Channels[branchId]
}

func getChannelsForFanIn(fanInId DagNodeId) map[int]chan map[DagNodeId]interface{} {
	return usedChannels[fanInId].Channels
}

func getOutputChannelForFanIn(fanInId DagNodeId) chan map[string]interface{} {
	return usedChannels[fanInId].OutputChannel
}

func clearChannelForFanIn(fanInId DagNodeId) {
	delete(usedChannels, fanInId)
}

/*
How the fan wait for previous output works:
- [v] who should hold the channel(s)? Fan-in
- [v] when initialize the channel(s)? when constructing the fan-in, but we need the branchNumbers
- [v] how should fan-in pass the channel? Providing a getChannelForParallelBranch(branchId) that return the corresponding channel for that branch
- [v] when should a node use the getChannelForParallelBranch method and send the result? Only when the next node is a Fan-In node, when passing output.
- [v] who should send to the channel(s)? The terminal node before the fan in each parallel branch
- [ ] when should send to the channel(s)? After the execution of the terminal node in each parallel branch
- [ ] who should receive from the channel(s)? This node, fan in.
- [ ] when should receive from the channel(s)? In this function, Exec.
*/

var DefaultTimeout = 60 * time.Second

func NewFanInNode(mergeMode MergeMode, fanInDegree int, branchNumbers []int, nillableTimeout *time.Duration) *FanInNode {
	timeout := nillableTimeout
	if timeout == nil {
		timeout = &DefaultTimeout
	}
	fanIn := FanInNode{
		Id:          DagNodeId(shortuuid.New()),
		NodeType:    FanIn,
		OutputTo:    "",
		FanInDegree: fanInDegree,
		Timeout:     *timeout,
		Mode:        mergeMode,
		//IsReached:   false,
	}
	createChannels(fanIn.Id, fanInDegree, branchNumbers)

	return &fanIn
}

func (f *FanInNode) Equals(cmp types.Comparable) bool {
	switch f1 := cmp.(type) {
	case *FanInNode:
		return f.Id == f1.Id && f.FanInDegree == f1.FanInDegree && f.OutputTo == f1.OutputTo &&
			f.Timeout == f1.Timeout && f.Mode == f1.Mode
	default:
		return false
	}
}

// Exec already have all inputs when executing, so it simply merges them with the chosen policy
func (f *FanInNode) Exec(compRequest *CompositionRequest) (map[string]interface{}, error) {
	t0 := time.Now()

	fanInOutput := make(map[string]interface{})
	if f.Mode == AddNewMapEntry { // each map entry should have a different name map[i: map[nameI: valueI]]
		duplicates := make(map[string]int)
		for _, inputMap := range f.input {
			for name, value := range inputMap {
				num, ok := duplicates[name]
				duplicates[name] += 1
				if !ok {
					fanInOutput[name] = value
				} else {
					fanInOutput[fmt.Sprintf("%s_%d", name, num)] = value
				}

			}
		}
	} else if f.Mode == AddToArrayEntry { // all input maps MUST have exactly one entry with the same name
		valid := true
		name := ""
		for _, inputMap := range f.input {
			if len(inputMap) != 1 {
				return nil, fmt.Errorf("fanIn input map does not have 1 element")
			}
			for k, value := range inputMap {
				if name == "" {
					name = k
					fanInOutput[name] = make([]interface{}, 0)
				} else if name != k {
					valid = false
					break
				}
				fanInOutput[name] = append(fanInOutput[name].([]interface{}), value)
			}
			if valid == false {
				return nil, fmt.Errorf("each fanIn input map must have the same name")
			}
		}
	} else if f.Mode == AddToSetEntry {
		for _, inputMap := range f.input {
			for name, value := range inputMap {
				_, found := fanInOutput[name]
				if !found {
					fanInOutput[name] = value
				}
			}
		}
	}

	respAndDuration := time.Now().Sub(t0).Seconds()
	compRequest.ExecReport.Reports[f.Id] = &function.ExecutionReport{
		Result:         fmt.Sprintf("%v", fanInOutput),
		ResponseTime:   respAndDuration,
		IsWarmStart:    true, // not in a container
		InitTime:       0,
		OffloadLatency: 0,
		Duration:       respAndDuration,
		SchedAction:    "",
	}
	return fanInOutput, nil
}

func (f *FanInNode) AddOutput(dag *Dag, dagNode DagNodeId) error {
	f.OutputTo = dagNode
	return nil
}

// ReceiveInput simply saves the input map of each previous node into an array of them. Can fail if the input array ends having more maps then fanInDegree
func (f *FanInNode) ReceiveInput(input map[string]interface{}) error {
	if f.input == nil {
		f.input = make([]map[string]interface{}, 0)
	}
	f.input = append(f.input, input)

	if len(f.input) > f.FanInDegree {
		return fmt.Errorf("fan in has more input (%d) than its fanInDegree (%d). Terminating workflow", len(f.input), f.FanInDegree)
	}

	return nil
}

func (f *FanInNode) PrepareOutput(dag *Dag, output map[string]interface{}) error {
	return nil // we should not do nothing, the output should be already ok
}

func (f *FanInNode) GetNext() []DagNodeId {
	// we only have one output
	arr := make([]DagNodeId, 1)
	if f.OutputTo == "" {
		panic("you forgot to initialize OutputTo for FanInNode")
	}
	arr[0] = f.OutputTo
	return arr
}

func (f *FanInNode) Width() int {
	return f.FanInDegree
}

func (f *FanInNode) Name() string {
	return "Fan In"
}

func (f *FanInNode) ToString() string {
	return fmt.Sprintf("[FanInNode(%d)]", f.FanInDegree)
}

func (f *FanInNode) setBranchId(number int) {
	f.BranchId = number
}
func (f *FanInNode) GetBranchId() int {
	return f.BranchId
}

func (f *FanInNode) GetId() DagNodeId {
	return f.Id
}
