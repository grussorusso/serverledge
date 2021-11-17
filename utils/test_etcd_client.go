package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/context"
	"io/ioutil"
	"time"
)

/**
Simple example to understand etcd apis.
This can push a function to the etcd store inside the container.
N.B. In order to store the desired code we need to base64 encoding it because only string and byte arrays are admitted.
*/
func main() {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		// handle error!
	}
	defer cli.Close()
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	var raw map[string]interface{}
	b, err := ioutil.ReadFile("examples/python310.tar")
	if err != nil {
		fmt.Print(err)
	}
	encoded := base64.StdEncoding.EncodeToString(b)
	// store all the info  as a json string
	fieldMap := map[string]string{"runtime": "python310", "memory": "768", "handler": "function.handler", "code": encoded}
	jsonStr, _ := json.Marshal(fieldMap)

	json.Unmarshal(jsonStr, &raw)
	//insert a code (value) inside the etcd store associated with his name ( key )
	cli.Put(ctx, "app", string(jsonStr))

	// test: getting back the application code
	getresp, err := cli.Get(ctx, "app")

	print(string(getresp.Kvs[0].Value))
	// use the response

}
