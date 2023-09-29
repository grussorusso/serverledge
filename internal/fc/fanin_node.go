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
	input       map[string]interface{}
	IsReached   bool
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
		IsReached:   false,
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

// Exec waits all output from previous nodes or return an error after a timeout expires
func (f *FanInNode) Exec(compRequest *CompositionRequest) (map[string]interface{}, error) {
	if !f.IsReached {
		f.IsReached = true
	} else {
		return nil, nil // fmt.Errorf("node is already reached, skip me")
	}
	t0 := time.Now()
	okChan := make(chan bool)
	// getting outputs
	go func() {
		outputs := make(map[int]map[DagNodeId]interface{})
		channels := getChannelsForFanIn(f.Id)
		for br, ch := range channels {
			outputs[br] = <-ch // TODO: no one send to this channel!!!
		}
		fmt.Println("retrieved all inputs")
		okChan <- true
		fanInOutput := make(map[string]interface{})
		if f.Mode == OneMapEntryForEachBranch {
			for i, outMap := range outputs {
				for name, value := range outMap {
					fanInOutput[fmt.Sprintf("%s_%d", name, i)] = value
				}
			}
		} else if f.Mode == OneMapArrayEntryForEachBranch {
			fmt.Println("OneMapArrayEntryForEachBranch not implemented")
			okChan <- false
			return
		}
		fanInOutputChannel := getOutputChannelForFanIn(f.Id)
		fanInOutputChannel <- fanInOutput

	}()
	// implementing timeout
	cancel := time.AfterFunc(f.Timeout, func() {
		fmt.Println("timeout elapsed")
		okChan <- false
	})

	ok := <-okChan
	var output map[string]interface{} = nil
	var err error = nil
	if ok {
		cancel.Stop() // stopping timer, we're all good
		// merging outputs with the chosen mergeMode
		fanInOutputChannel := getOutputChannelForFanIn(f.Id)
		output = <-fanInOutputChannel
	} else {
		err = fmt.Errorf("fan-in merge failed - timeout occurred")
	}
	respAndDuration := time.Now().Sub(t0).Seconds()
	compRequest.ExecReport.Reports[f.Id] = &function.ExecutionReport{
		Result:         fmt.Sprintf("%v", output),
		ResponseTime:   respAndDuration,
		IsWarmStart:    true, // not in a container
		InitTime:       0,
		OffloadLatency: 0,
		Duration:       respAndDuration,
		SchedAction:    "",
	}
	return output, err
}

func (f *FanInNode) AddOutput(dag *Dag, dagNode DagNodeId) error {
	//if f.OutputTo != nil {
	//	return errors.New("result already present in node")
	//}

	f.OutputTo = dagNode
	return nil
}

func (f *FanInNode) ReceiveInput(input map[string]interface{}) error {
	// TODO: devi ricevere gli input separatamente dai nodi precedenti.
	f.input = input
	return nil
}

func (f *FanInNode) PrepareOutput(dag *Dag, output map[string]interface{}) error {
	return nil // we should not do nothing, the output should be already ok
}

func (f *FanInNode) GetNext() []DagNodeId {
	// we only have one output
	// TODO: we should wait for function to complete!
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
