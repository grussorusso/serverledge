package scheduling

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func publishAsyncResponse(reqId string, response function.Response) {
	etcdClient, err := utils.GetEtcdClient()
	if err != nil {
		log.Fatal("Client not available")
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)

	resp, err := etcdClient.Grant(ctx, 1800)
	if err != nil {
		log.Fatal(err)
		return
	}

	key := fmt.Sprintf("async/%s", reqId)
	payload, err := json.Marshal(response)
	if err != nil {
		log.Printf("Could not marshal response: %v", err)
		return
	}

	_, err = etcdClient.Put(ctx, key, string(payload), clientv3.WithLease(resp.ID))
	if err != nil {
		log.Fatal(err)
		return
	}

	//res, err := etcdClient.Get(ctx, key)
	//if err != nil {
	//	log.Fatal(err)
	//	return
	//}
	//payload = res.Kvs[0].Value
	//var newResp function.Response
	//json.Unmarshal(payload, &newResp)
	//log.Printf("%v", newResp)
}
