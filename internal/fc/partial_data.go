package fc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/grussorusso/serverledge/internal/cache"
	"github.com/grussorusso/serverledge/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

type PartialDataId string

func newPartialDataId(reqId ReqId) PartialDataId {
	return PartialDataId("partialData_" + reqId)
}

// PartialData is saved separately from progressData to avoid cluttering the Progress struct and each Serverledge node's cache
type PartialData struct {
	ReqId    ReqId     // request referring to this partial data
	ForNode  DagNodeId // dagNode that should receive this partial data
	FromNode DagNodeId // useful for fanin
	Data     map[string]interface{}
}

func (pd PartialData) Equals(pd2 *PartialData) bool {

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

func getPartialDataEtcdKey(reqId ReqId, nodeId DagNodeId) string {
	return fmt.Sprintf("/partialData/%s/%s", reqId, nodeId)
}

func SavePartialData(pd *PartialData) error {
	err := savePartialDataToEtcd(pd)
	if err != nil {
		return err
	}
	inCache := savePartialDataInCache(pd)
	if !inCache {
		return errors.New("failed to save partialData in cache")
	}
	return nil
}

func RetrievePartialData(reqId ReqId, nodeId DagNodeId) ([]*PartialData, bool) {
	var err error

	// Get from cache if exists, otherwise from ETCD
	partialDatas, found := getPartialDataFromCache(newPartialDataId(reqId), nodeId)
	if !found {
		// cache miss - retrieve partialData from ETCD
		partialDatas, err = getPartialDataFromEtcd(reqId, nodeId)
		if err != nil {
			return nil, false
		}
		// insert a new element to the cache
		ok := savePartialDataInCache(partialDatas...)
		if !ok {
			return nil, false
		}
	}

	return partialDatas, true
}

func RetrieveSinglePartialData(reqId ReqId, nodeId DagNodeId) (*PartialData, error) {
	pds, found := RetrievePartialData(reqId, nodeId)
	if !found {
		return nil, fmt.Errorf("partial data not found")
	} else if len(pds) == 0 {
		return nil, fmt.Errorf("partial data are empty")
	} else if len(pds) > 1 {
		return nil, fmt.Errorf("more than one partial data for a simple node")
	}
	return pds[0], nil
}

// RetrieveAllPartialData returns all partial data associated with a request
func RetrieveAllPartialData(reqId ReqId) (map[DagNodeId][]*PartialData, error) {
	partialDataMap := make(map[DagNodeId][]*PartialData)

	partialDataFromCache := getAllPartialDataFromCache(newPartialDataId(reqId))

	for dagNodeId, data := range partialDataFromCache {
		partialDataMap[dagNodeId] = data
	}

	partialDataFromEtcd, err := getAllPartialDataFromEtcd(reqId)
	if err == nil {
		for dagNodeId, data := range partialDataFromEtcd {
			partialDataMap[dagNodeId] = data
		}
	} else {
		return nil, err
	}

	return partialDataMap, nil
}

func NumberOfPartialDataFor(reqId ReqId) int {
	partialDataMap := make(map[DagNodeId][]*PartialData)

	partialDataFromCache := getAllPartialDataFromCache(newPartialDataId(reqId))

	for dagNodeId, data := range partialDataFromCache {
		partialDataMap[dagNodeId] = data
	}

	partialDataFromEtcd, err := getAllPartialDataFromEtcd(reqId)
	if err == nil {
		for dagNodeId, data := range partialDataFromEtcd {
			partialDataMap[dagNodeId] = data
		}
	}

	return len(partialDataMap)
}

// FIXME: seems useless
//func DeletePartialData(reqId ReqId, nodeId DagNodeId) error {
//	cli, err := utils.GetEtcdClient()
//	if err != nil {
//		return fmt.Errorf("failed to connect to etcd: %v", err)
//	}
//	ctx := context.TODO()
//	// remove the progress from ETCD
//	dresp, err := cli.Delete(ctx, getPartialDataEtcdKey(reqId, nodeId))
//	if err != nil || dresp.Deleted != 1 {
//		return fmt.Errorf("failed partialData delete: %v", err)
//	}
//
//	// pdFromCache, found := getPartialDataFromCache(reqId, nodeId)
//	// Remove the progress from the local cache // FIXME: rimuovi solo quello con nodeId specificato!
//	cache.GetCacheInstance().Delete(string(reqId))
//
//	return nil
//}

func DeleteAllPartialData(reqId ReqId) (int64, error) {
	cli, err := utils.GetEtcdClient()
	if err != nil {
		return 0, fmt.Errorf("failed to connect to etcd: %v", err)
	}
	ctx := context.TODO()
	// remove the progress from ETCD
	removed, err := cli.Delete(ctx, getPartialDataEtcdKey(reqId, ""), clientv3.WithPrefix())
	if err != nil {
		return 0, fmt.Errorf("failed partialData delete: %v", err)
	}

	// Remove the progress from the local cache
	cache.GetCacheInstance().Delete(string(newPartialDataId(reqId)))
	return removed.Deleted, nil
}

// savePartialDataInCache appends in cache a partial data related to a specific request and dagNode in a Dag
func savePartialDataInCache(pds ...*PartialData) bool {
	c := cache.GetCacheInstance()
	for _, pd := range pds {
		partialDataIdType := newPartialDataId(pd.ReqId)
		partialDataId := string(partialDataIdType)
		if _, found := c.Get(partialDataId); !found {
			// we store an array of PartialData because fanIn can have multiple inputs
			partialDataMultiMap := make(map[PartialDataId]map[DagNodeId][]*PartialData)
			c.Set(partialDataId, partialDataMultiMap, cache.NoExpiration)
		}

		partialDataMap, found := c.Get(partialDataId)
		if !found {
			return false
		}

		typedPartialDataMap := partialDataMap.(map[PartialDataId]map[DagNodeId][]*PartialData)
		if typedPartialDataMap[partialDataIdType] == nil {
			typedPartialDataMap[partialDataIdType] = make(map[DagNodeId][]*PartialData)
		}
		_, foundSlice := typedPartialDataMap[partialDataIdType][pd.ForNode]
		if !foundSlice {
			typedPartialDataMap[partialDataIdType][pd.ForNode] = make([]*PartialData, 0)
		}
		typedPartialDataMap[partialDataIdType][pd.ForNode] = append(typedPartialDataMap[partialDataIdType][pd.ForNode], pd)
	}
	return true
}

func savePartialDataToEtcd(pd *PartialData) error {
	cli, err := utils.GetEtcdClient()
	if err != nil {
		return err
	}
	ctx := context.TODO()
	// marshal the progress object into json
	payload, err := json.Marshal(pd)
	if err != nil {
		return fmt.Errorf("could not marshal progress: %v", err)
	}
	// saves the json object into etcd
	_, err = cli.Put(ctx, getPartialDataEtcdKey(pd.ReqId, pd.ForNode), string(payload))
	if err != nil {
		return fmt.Errorf("failed etcd Put partial data: %v", err)
	}
	return nil
}

func getPartialDataFromCache(requestId PartialDataId, nodeId DagNodeId) ([]*PartialData, bool) {
	c := cache.GetCacheInstance()
	partialDataMap, found := c.Get(string(requestId))
	if !found {
		return nil, false
	}
	return partialDataMap.(map[PartialDataId]map[DagNodeId][]*PartialData)[requestId][nodeId], found
}

func getPartialDataFromEtcd(requestId ReqId, nodeId DagNodeId) ([]*PartialData, error) {
	cli, err := utils.GetEtcdClient()
	if err != nil {
		return nil, errors.New("failed to connect to ETCD")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	key := getPartialDataEtcdKey(requestId, nodeId)
	getResponse, err := cli.Get(ctx, key)
	if err != nil || len(getResponse.Kvs) < 1 {
		return nil, fmt.Errorf("failed to retrieve partialDatas for requestId: %s", key)
	}

	var partialDatas []*PartialData
	err = json.Unmarshal(getResponse.Kvs[0].Value, &partialDatas)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal partialDatas json: %v", err)
	}

	return partialDatas, nil
}

func getAllPartialDataFromEtcd(requestId ReqId) (map[DagNodeId][]*PartialData, error) {
	cli, err := utils.GetEtcdClient()
	if err != nil {
		return nil, errors.New("failed to connect to ETCD")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	key := getPartialDataEtcdKey(requestId, "") // we get only the request prefix, so we retrieve all data
	partialDataResponse, err := cli.Get(ctx, key, clientv3.WithPrefix())
	if err != nil || len(partialDataResponse.Kvs) < 1 {
		return nil, fmt.Errorf("failed to retrieve partialDataMap for requestId: %s", key)
	}

	partialDataMap := make(map[DagNodeId][]*PartialData)
	for _, kv := range partialDataResponse.Kvs {
		var partialData *PartialData
		err = json.Unmarshal(kv.Value, &partialData)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal partialDataMap json: %v", err)
		}
		_, found := partialDataMap[partialData.ForNode]
		if !found {
			partialDataMap[partialData.ForNode] = make([]*PartialData, 0)
		}
		partialDataMap[partialData.ForNode] = append(partialDataMap[partialData.ForNode], partialData)
	}

	return partialDataMap, nil
}

func getAllPartialDataFromCache(requestId PartialDataId) map[DagNodeId][]*PartialData {
	c := cache.GetCacheInstance()
	partialDataMap, found := c.Get(string(requestId))
	if !found {
		return make(map[DagNodeId][]*PartialData)
	}
	return partialDataMap.(map[PartialDataId]map[DagNodeId][]*PartialData)[requestId]
}
