package fc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cornelk/hashmap"
	"github.com/grussorusso/serverledge/internal/cache"
	"github.com/grussorusso/serverledge/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
	"sync"
	"time"
)

type PartialDataId string

var pdCacheMutex = &sync.Mutex{}
var pdSliceMutex = &sync.Mutex{}

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

var pdCache = make(map[PartialDataId]map[DagNodeId][]*PartialData)

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

func SavePartialData(pd *PartialData, saveAlsoOnEtcd bool) error {
	if saveAlsoOnEtcd {
		err := savePartialDataToEtcd(pd)
		if err != nil {
			return err
		}
	}
	inCache := savePartialDataInCache(pd)
	if !inCache {
		return errors.New("failed to save partialData in cache")
	}
	return nil
}

func RetrievePartialData(reqId ReqId, nodeId DagNodeId) ([]*PartialData, error) {
	// Get from cache if exists, otherwise from ETCD
	partialDatas, err := getPartialDataFromCache(newPartialDataId(reqId), nodeId)
	if err != nil {
		fmt.Printf("cache miss: %v\n", err)
		// cache miss - retrieve partialData from ETCD
		partialDatas, err = getPartialDataFromEtcd(reqId, nodeId)
		if err != nil {
			return nil, fmt.Errorf("partial data not found in cache and in etcd: %v\n", err)
		}
		// insert a new element to the cache
		ok := savePartialDataInCache(partialDatas...)
		if !ok {
			return nil, fmt.Errorf("failed to save in cache a found partial data")
		}
	}
	if len(partialDatas) == 0 {
		return nil, fmt.Errorf("partial data are empty")
	}

	return partialDatas, nil
}

func RetrieveSinglePartialData(reqId ReqId, nodeId DagNodeId) (*PartialData, error) {
	pds, err := RetrievePartialData(reqId, nodeId)
	if err != nil {
		return nil, fmt.Errorf("partial data not found: %v", err)
	} else if len(pds) > 1 {
		return nil, fmt.Errorf("more than one partial data for a simple node")
	}
	return pds[0], nil
}

// RetrieveAllPartialData returns all partial data associated with a request
func RetrieveAllPartialData(reqId ReqId, alsoFromEtcd bool) (*hashmap.Map[DagNodeId, []*PartialData], error) {
	partialDataMap := hashmap.New[DagNodeId, []*PartialData]()

	partialDataFromCache := getAllPartialDataFromCache(newPartialDataId(reqId))

	partialDataFromCache.Range(func(dagNodeId DagNodeId, data []*PartialData) bool {
		partialDataMap.Set(dagNodeId, data)
		return true
	})
	if alsoFromEtcd {
		partialDataFromEtcd, err := getAllPartialDataFromEtcd(reqId)
		if err == nil {
			for dagNodeId, data := range partialDataFromEtcd {
				partialDataMap.Set(dagNodeId, data)
			}
		} else {
			return nil, err
		}
	}

	return partialDataMap, nil
}

func NumberOfPartialDataFor(reqId ReqId) int {
	partialDataMap := hashmap.New[DagNodeId, []*PartialData]()

	partialDataFromCache := getAllPartialDataFromCache(newPartialDataId(reqId))

	partialDataFromCache.Range(func(dagNodeId DagNodeId, data []*PartialData) bool {
		partialDataMap.Set(dagNodeId, data)
		return true
	})

	partialDataFromEtcd, err := getAllPartialDataFromEtcd(reqId)
	if err == nil {
		for dagNodeId, data := range partialDataFromEtcd {
			partialDataMap.Set(dagNodeId, data)
		}
	}

	return partialDataMap.Len()
}

func DeleteAllPartialData(reqId ReqId) (int64, error) {
	cli, err := utils.GetEtcdClient()
	if err != nil {
		return 0, fmt.Errorf("failed to connect to etcd: %v", err)
	}
	ctx := context.TODO()

	pdCacheMutex.Lock()
	// Remove the partial data from the local cache
	// cache.GetCacheInstance().Delete(string(newPartialDataId(reqId)))
	delete(pdCache, newPartialDataId(reqId))
	// remove the partial data from ETCD
	removed, err := cli.Delete(ctx, getPartialDataEtcdKey(reqId, ""), clientv3.WithPrefix())
	if err != nil {
		pdCacheMutex.Unlock()
		return 0, fmt.Errorf("failed partialData delete: %v", err)
	}
	pdCacheMutex.Unlock()

	return removed.Deleted, nil
}

// savePartialDataInCache appends in cache a partial data related to a specific request and dagNode in a Dag
func savePartialDataInCache(pds ...*PartialData) bool {
	//c := cache.GetCacheInstance()
	pdCacheMutex.Lock()
	pdCacheMutex.Unlock()
	for _, pd := range pds {
		partialDataIdType := newPartialDataId(pd.ReqId)
		//partialDataId := string(partialDataIdType)
		pdSliceMutex.Lock()
		partialDataMap, found /*isLoaded*/ := pdCache[partialDataIdType]
		if !found {
			partialDataMap = make(map[DagNodeId][]*PartialData)
		}
		//if !found {
		// we store an array of PartialData because fanIn can have multiple inputs
		// partialDataMap = hashmap.New[PartialDataId, *hashmap.Map[DagNodeId, []*PartialData]]()
		// return false
		//}
		slice, foundSlice := partialDataMap[pd.ForNode]
		if !foundSlice {
			slice = make([]*PartialData, 0)
		}
		slice = append(slice, pd)
		partialDataMap[pd.ForNode] = slice
		pdCache[partialDataIdType] = partialDataMap
		pdSliceMutex.Unlock()
		//typedPartialDataMap := partialDataMap.(*hashmap.Map[PartialDataId, *hashmap.Map[DagNodeId, []*PartialData]])
		//subMap, isLoaded := typedPartialDataMap.GetOrInsert(partialDataIdType, hashmap.New[DagNodeId, []*PartialData]())
		//if isLoaded {
		//	if subMap.Len() == 0 {
		//		return false
		//	}
		//}
		//pdSlice, isLoaded2 := subMap.GetOrInsert(pd.ForNode, make([]*PartialData, 0))
		//if !isLoaded2 {
		//	if pdSlice == nil {
		//		return false
		//	}
		//}
		//pdSlice = append(pdSlice, pd)
		//subMap.Set(pd.ForNode, pdSlice)
		//typedPartialDataMap.Set(partialDataIdType, subMap)
		//c.Set(partialDataId, typedPartialDataMap, cache.NoExpiration)
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
	pdCacheMutex.Lock()
	defer pdCacheMutex.Unlock()
	// saves the json object into etcd
	_, err = cli.Put(ctx, getPartialDataEtcdKey(pd.ReqId, pd.ForNode), string(payload))
	if err != nil {
		return fmt.Errorf("failed etcd Put partial data: %v", err)
	}
	return nil
}

func getPartialDataFromCache(pdId PartialDataId, nodeId DagNodeId) ([]*PartialData, error) {
	// c := cache.GetCacheInstance()
	// getting first the entire map from cache
	//partialDataMap, found := c.Get(string(pdId))
	//if !found {
	//	return nil, fmt.Errorf("cannot find partial data map in cache for request id %s\n", pdId)
	//}
	// casting map
	// typedPartialDataMap, ok := partialDataMap.(*hashmap.Map[PartialDataId, *hashmap.Map[DagNodeId, []*PartialData]])
	// if !ok {
	// 	return nil, fmt.Errorf("failed to cast interface to hashmap of partial datas")
	// }
	// getting the sub map
	pdCacheMutex.Lock()
	defer pdCacheMutex.Unlock()
	pdSliceMutex.Lock()
	subMap, ok := pdCache[pdId]
	if !ok {
		pdSliceMutex.Unlock()
		return nil, fmt.Errorf("cannot find partial data submap for request id %s\n", pdId)
	}
	pdSliceMutex.Unlock()
	// getting the slice
	slice, b := subMap[nodeId]
	if !b {
		return nil, fmt.Errorf("cannot find slice of partial data for request id %s and dag node %s\n", pdId, nodeId)
	}
	return slice, nil
}

func getPartialDataFromEtcd(requestId ReqId, nodeId DagNodeId) ([]*PartialData, error) {
	cli, err := utils.GetEtcdClient()
	if err != nil {
		return nil, errors.New("failed to connect to ETCD")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	key := getPartialDataEtcdKey(requestId, nodeId)
	println("getting key from etcd: ", key)
	pdCacheMutex.Lock()
	getResponse, err := cli.Get(ctx, key)
	if err != nil || len(getResponse.Kvs) < 1 {
		pdCacheMutex.Unlock()
		return nil, fmt.Errorf("failed to retrieve partialDatas for requestId: %s", key)
	}
	pdCacheMutex.Unlock()
	partialDatas := make([]*PartialData, 0, len(getResponse.Kvs))
	for _, v := range getResponse.Kvs {
		var partialData *PartialData
		err = json.Unmarshal(v.Value, &partialData)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal partialDatas json: %v", err)
		}
		partialDatas = append(partialDatas, partialData)
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
	pdCacheMutex.Lock()
	partialDataResponse, err := cli.Get(ctx, key, clientv3.WithPrefix())
	if err != nil || len(partialDataResponse.Kvs) < 1 {
		pdCacheMutex.Unlock()
		return nil, fmt.Errorf("failed to retrieve partialDataMap for requestId: %s", key)
	}
	pdCacheMutex.Unlock()

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

func getAllPartialDataFromCache(requestId PartialDataId) *hashmap.Map[DagNodeId, []*PartialData] {
	c := cache.GetCacheInstance()
	partialDataMap, found := c.Get(string(requestId))
	if !found {
		return hashmap.New[DagNodeId, []*PartialData]()
	}
	aaa, ok := partialDataMap.(*hashmap.Map[PartialDataId, *hashmap.Map[DagNodeId, []*PartialData]]).Get(requestId)
	if !ok {
		return hashmap.New[DagNodeId, []*PartialData]()
	}
	return aaa
}
