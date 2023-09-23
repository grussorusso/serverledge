package fc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/grussorusso/serverledge/internal/cache"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/types"
	"github.com/grussorusso/serverledge/utils"
	"github.com/labstack/gommon/log"
	"golang.org/x/exp/slices"
	"time"
)

// FunctionComposition is a serverless Function Composition
type FunctionComposition struct {
	Name      string // al posto del nome potrebbe essere un id da mettere in etcd
	Functions map[string]*function.Function
	Workflow  Dag
	// Progress           *Progress
	RemoveFnOnDeletion bool // TODO: spostare dentro la request
	// ExecReport ExecutionReport
}

type ExecutionReport struct {
	Result       map[string]interface{}
	reports      []function.ExecutionReport
	ResponseTime float64 // time waited by the user to get the output of the entire composition
	// InitTime       float64 // time spent sleeping before executing the request (the cold start)
	// OffloadLatency float64 // time spent offloading the requests
	// Duration       float64 // time spent executing the requests
}

// NewFC instantiates a new FunctionComposition with a name and a corresponding dag. Function can contain duplicate functions (with the same name)
func NewFC(name string, dag Dag, functions []*function.Function, removeFnOnDeletion bool) FunctionComposition {
	functionMap := make(map[string]*function.Function)
	for _, f := range functions {
		// if the function is already added, simply replace it
		functionMap[f.Name] = f
	}

	return FunctionComposition{
		Name:               name,
		Functions:          functionMap,
		Workflow:           dag,
		RemoveFnOnDeletion: removeFnOnDeletion,
		// ExecReport: ExecutionReport{},
	}
}

func (fc *FunctionComposition) getEtcdKey() string {
	return getEtcdKey(fc.Name)
}

func getEtcdKey(fc string) string {
	return fmt.Sprintf("/fc/%s", fc)
}

// GetAllFC returns the function composition names
func GetAllFC() ([]string, error) {
	return function.GetAllWithPrefix("/fc")
}

// FIXME: this should return Deployable and be merged with function.getFromCache
func getFCFromCache(name string) (*FunctionComposition, bool) {
	localCache := cache.GetCacheInstance()
	cachedObj, found := localCache.Get(name)
	if !found {
		return nil, false
	}
	//cache hit
	//return a safe copy of the function composition previously obtained
	fc := *cachedObj.(*FunctionComposition)
	return &fc, true
}

// FIXME: this should return Deployable and merged with function.getFromEtcd
func getFCFromEtcd(name string) (*FunctionComposition, error) {
	cli, err := utils.GetEtcdClient()
	if err != nil {
		return nil, errors.New("failed to connect to ETCD")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	getResponse, err := cli.Get(ctx, getEtcdKey(name))
	if err != nil || len(getResponse.Kvs) < 1 {
		return nil, errors.New("failed to retrieve value for key")
	}

	var f FunctionComposition
	err = json.Unmarshal(getResponse.Kvs[0].Value, &f)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal json: %v", err)
	}

	return &f, nil
}

// FIXME: this shold return Deployable and be merged with function.GetFunction
func GetFC(name string) (*FunctionComposition, bool) {
	val, found := getFCFromCache(name)
	if !found {
		// cache miss
		f, err := getFCFromEtcd(name)
		if err != nil {
			log.Error(err.Error())
			return nil, false
		}
		//insert a new element to the cache
		cache.GetCacheInstance().Set(name, f, cache.DefaultExp)
		return f, true
	}

	return val, true
}

// SaveToEtcd creates and register the function composition in Serverledge
// It is like SaveToEtcd for a simple function
// TODO: maybe we should merge with *function.SaveToEtcd and use a Deployable as argument
// TODO: maybe we should register all function defined in the DAG
func (fc *FunctionComposition) SaveToEtcd() error {
	cli, err := utils.GetEtcdClient()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	// Save all functions in the dag to ETCD
	// funcs := make([]*function.Function, 0)
	for _, fName := range fc.Workflow.GetUniqueDagFunctions() {
		_, exists := function.GetFunction(fName)
		if !exists {
			errSave := fc.Functions[fName].SaveToEtcd()
			if errSave != nil {
				return fmt.Errorf("failed to save function %s: %v", fName, errSave)
			}
		}
		// funcs = append(funcs, f)
	}

	// marshal the function composition object into json
	payload, err := json.Marshal(*fc)
	if err != nil {
		return fmt.Errorf("could not marshal function composition: %v", err)
	}
	// saves the json object into etcd
	_, err = cli.Put(ctx, fc.getEtcdKey(), string(payload))
	if err != nil {
		return fmt.Errorf("failed etcd Put: %v", err)
	}

	// Add the function composition to the local cache
	cache.GetCacheInstance().Set(fc.Name, fc, cache.DefaultExp)

	return nil
}

// Invoke schedules each function of the composition and invokes them
func (fc *FunctionComposition) Invoke(input map[string]interface{}) (ExecutionReport, error) {
	// fc.Progress.Retrieve()

	output, err := fc.Workflow.Execute(input)
	// fc.ExecReport.Result = output
	return ExecutionReport{Result: output}, err
}

// Delete removes the FunctionComposition from cache and from etcd, so it cannot be invoked anymore
func (fc *FunctionComposition) Delete() error {
	cli, err := utils.GetEtcdClient()
	if err != nil {
		return err
	}
	ctx := context.TODO()
	if fc.RemoveFnOnDeletion {
		for _, f := range fc.Functions {
			err := f.Delete()
			if err != nil {
				return fmt.Errorf("failed to delete function %s associated to function composition %s: %v", f.Name, fc.Name, err)
			}
		}
	}

	dresp, err := cli.Delete(ctx, fc.getEtcdKey())
	if err != nil || dresp.Deleted != 1 {
		return fmt.Errorf("failed Delete: %v", err)
	}

	// Remove the function from the local cache
	cache.GetCacheInstance().Delete(fc.Name)

	return nil
}

// DeleteAll deletes the function composition from Etcd and the Functions associated with it
func (fc *FunctionComposition) DeleteAll() error {
	err := fc.Delete()

	for _, fName := range fc.Workflow.GetUniqueDagFunctions() {
		f, exists := function.GetFunction(fName)
		if !exists {
			return fmt.Errorf("funtion %s does not exist", fName)
		}
		err1 := f.Delete()
		if err1 != nil {
			return fmt.Errorf("the deletion of the function %s has failed", f.Name)
		}
	}

	return err
}

func (fc *FunctionComposition) Poll() interface{} {
	panic("implement me")
}

// Equals is used in tests to check function composition equality
func (fc *FunctionComposition) Equals(cmp types.Comparable) bool {
	fc2 := cmp.(*FunctionComposition)
	if fc.Name != fc2.Name {
		return false
	}
	if !slices.Equal(fc.Workflow.GetUniqueDagFunctions(), fc2.Workflow.GetUniqueDagFunctions()) {
		return false
	}
	if !fc.Workflow.Equals(&fc2.Workflow) {
		return false
	}
	return true
}
