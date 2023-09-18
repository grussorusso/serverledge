package fc

import "fmt"

// Progress tracks the progress of a Dag, i.e. which nodes are executed, and what is the next node to run. Dag progress is saved in ETCD and retrieved by the next node
type Progress struct {
	reqId        string // requestId, used to distinguish different dag's progresses
	totalNodes   int    // total number of Nodes to execute. Unknown if there is at least a choice node
	doneNodes    int    // number of already completed node
	nextNodeId   string // id of next dagNode to execute
	nextInputRef string // id of input data for the next node
}

// PartialData is saved separately from progressData to avoid cluttering the Progress struct and each Serverledge node's cache
type PartialData struct {
	ReqId     string // request referring to this partial data
	DagNodeId string // dagNode that should receive this partial data
	Data      map[string]interface{}
}

func InitProgress(dag *Dag) *Progress {
	return &Progress{
		totalNodes: len(dag.Nodes),
		doneNodes:  0,
		nextNodeId: dag.Start.Id,
	}
}

func (p *Progress) Print() {
	fmt.Printf("starting...")
	fmt.Printf("%d/%d", p.doneNodes, p.totalNodes)
	fmt.Printf("finishing...")
	fmt.Printf("completed")
}

// Update should be used by a completed node after its execution
func (p *Progress) Update() {
	p.doneNodes++ // TODO: how to deal with choice nodes?
}

// Save should be used by a completed node after its execution
func (p *Progress) Save(reqId string) {
	// TODO: save progress in ETCD
}

// Retrieve should be used by the next node to execute
func (p *Progress) Retrieve(reqId string) {
	// TODO: retrieve progress from ETCD
}

func (p *Progress) IsCompleted() bool {
	return p.totalNodes == p.doneNodes
}

func (pd *PartialData) Retrieve() (map[string]interface{}, error) {
	// TODO: if data is colocated in this Serverledge node, we should get data from here
	//  otherwise, retrieve data from ETCD
	return pd.Data, nil
}

func (pd *PartialData) Save() {
	// TODO: save data on ETCD
}

func (pd *PartialData) Purge() {
	// TODO: delete from etcd: all partial data connected to the same request should be deleted, only after the dag is complete.
}

// TODO: We should have a local cache for this data and progress!!!
