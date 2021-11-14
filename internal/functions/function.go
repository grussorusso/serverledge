package functions

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/grussorusso/serverledge/cache"
	"github.com/grussorusso/serverledge/internal/config"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/context"
	"strconv"
	"time"
)

//A serverless Function.
type Function struct {
	Name            string
	Runtime         string // example: python310
	Memory          int
	Handler         string // example: "module.function_name"
	TarFunctionCode string // input is .tar
}

//GetFunction retrieves a Function given its name.
func GetFunction(name string) (*Function, bool) {

	val, found := getFromCache(name)
	if !found {
		// cache miss
		fmt.Println("Cache miss!")
		f, response := getFromEtcd(name)
		if !response {
			return nil, false
		}
		//insert a new element to the cache
		cache.GetCacheInstance().Set(name, f, cache.DefaultExp)
		return f, true
	}

	//cache hit
	fmt.Println("Cache Hit")
	return val, true

	//return &Function{"prova", "python310", 256, "function.handler", "http://www.ce.uniroma2.it/~russorusso/python310.tar"}, true
}

func (f *Function) String() string {
	return f.Name
}

func getFromCache(name string) (*Function, bool) {
	localCache := cache.GetCacheInstance()
	f, found := localCache.Get(name)
	if !found {
		return nil, false
	}
	//cache hit
	return f.(*Function), true

}

func getFromEtcd(name string) (*Function, bool) {
	//etcd v3 client
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{config.GetString("etcd.address", "localhost:2379")},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, false
	}
	defer cli.Close()
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	// retrieve the application from etcd by using his name as a key
	getResponse, err := cli.Get(ctx, name)
	if err != nil {
		return nil, false
	}
	// function properties : returned value (json format)
	var jsonMap map[string]string
	err = json.Unmarshal(getResponse.Kvs[0].Value, &jsonMap)
	if err != nil {
		return nil, false
	}
	decoded, _ := base64.StdEncoding.DecodeString(jsonMap["code"])
	memory, _ := strconv.Atoi(jsonMap["memory"])

	return &Function{name, jsonMap["runtime"], memory, jsonMap["handler"], string(decoded)}, true
}
