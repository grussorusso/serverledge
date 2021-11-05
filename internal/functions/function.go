package functions

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/context"
	"strconv"
	"time"
)

//A serverless Function.
type Function struct {
	Name         string
	Runtime      string // example: python310
	Memory       int
	Handler      string // example: "module.function_name"
	FunctionCode string
}

//GetFunction retrieves a Function given its name.
func GetFunction(name string) (*Function, bool) {
	//TODO: info should be retrieved from the DB (or possibly through a
	//local cache)

	//etcd v3 client
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"}, // todo change this one
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		// handle error!
	}
	defer cli.Close()
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	// retrieve the application from etcd by using his name as a key
	getResponse, err := cli.Get(ctx, "app2") // todo fix this name
	// function properties : returned value (json format)
	var jsonMap map[string]string
	json.Unmarshal(getResponse.Kvs[0].Value, &jsonMap)
	decoded, _ := base64.StdEncoding.DecodeString(jsonMap["code"])
	memory, _ := strconv.Atoi(jsonMap["memory"])

	//AUDIT todo delete those lines
	fmt.Println(memory)
	fmt.Println(jsonMap["handler"])
	fmt.Println(jsonMap["runtime"])

	return &Function{"app2", jsonMap["runtime"], memory, jsonMap["handler"], string(decoded)}, true
	//return &Function{"prova", "python310", 256, "function.handler", "http://www.ce.uniroma2.it/~russorusso/python310.tar"}, true
}

func (f *Function) String() string {
	return f.Name
}
