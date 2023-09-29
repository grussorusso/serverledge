package fc

import (
	"fmt"
	"sync"
)

type DagNodeId string

var partialDataCache = newPartialDataCache()

// PartialData is saved separately from progressData to avoid cluttering the Progress struct and each Serverledge node's cache
type PartialData struct {
	ReqId    ReqId     // request referring to this partial data
	ForNode  DagNodeId // dagNode that should receive this partial data
	FromNode DagNodeId // useful for fanin
	Data     map[string]interface{}
}

func (pd PartialData) Equals(pd2 PartialData) bool {

	if len(pd.Data) != len(pd2.Data) {
		return false
	}

	for s := range pd.Data {
		// we convert the type to string to avoid checking all possible types!!!
		value1 := fmt.Sprintf("%v", pd.Data[s])
		value2 := fmt.Sprintf("%v", pd2.Data[s])
		if value1 != value2 {
			return false
		}
	}

	return pd.ReqId == pd2.ReqId && pd.FromNode == pd2.FromNode && pd.ForNode == pd2.ForNode
}

func (pd PartialData) String() string {
	return fmt.Sprintf(`PartialData{
		ReqId:    %s,
		ForNode:  %s,
		FromNode: %s,
		Data:     %v,
	}`, pd.ReqId, pd.ForNode, pd.FromNode, pd.Data)
}

func NewPartialData(reqId ReqId, forNode DagNodeId, fromNode DagNodeId, data map[string]interface{}) *PartialData {
	return &PartialData{
		ReqId:    reqId,
		ForNode:  forNode,
		FromNode: fromNode,
		Data:     data,
	}
}

type PartialDataCache struct {
	partialDatas map[ReqId]map[DagNodeId]*PartialData
	mutex        sync.RWMutex
}

func newPartialDataCache() PartialDataCache {
	return PartialDataCache{
		partialDatas: make(map[ReqId]map[DagNodeId]*PartialData),
	}
}

func (cache *PartialDataCache) InitNewRequest(req ReqId) {
	partialDataCache.partialDatas[req] = make(map[DagNodeId]*PartialData)
}

func (cache *PartialDataCache) Save(pd *PartialData) {
	// TODO: Save always in cache and in ETCD
	partialDataCache.partialDatas[pd.ReqId][pd.ForNode] = pd
}

func (cache *PartialDataCache) Retrieve(reqId ReqId, nodeId DagNodeId) (map[string]interface{}, error) {
	// TODO: if data is colocated in this Serverledge node, we should get data from here
	//  otherwise, retrieve data from ETCD
	requestPartialDatas, okReq := partialDataCache.partialDatas[reqId]
	if okReq {
		data, okDagNode := requestPartialDatas[nodeId]
		if okDagNode {
			return data.Data, nil
		} else {
			return nil, fmt.Errorf("failed to retrieve partial data for node %s", nodeId)
		}
	}
	return nil, fmt.Errorf("failed to retrieve partial datas for request %s", reqId)
}

func (cache *PartialDataCache) Purge(reqId ReqId) {
	delete(partialDataCache.partialDatas, reqId) // this should remove also the sub-map
	// TODO: delete from etcd: all partial data connected to the same request should be deleted, only after the dag is complete.
}

func IsEmptyPartialDataCache() bool {
	return len(partialDataCache.partialDatas) == 0
}
