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
	FromNode DagNodeId // TODO: maybe useless
	Data     map[string]interface{}
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
	partialDataCache.partialDatas[ReqId(req)] = make(map[DagNodeId]*PartialData)
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
	delete(progressCache.progresses, reqId) // this should remove also the sub-map
	// TODO: delete from etcd: all partial data connected to the same request should be deleted, only after the dag is complete.
}
