package scheduling

import (
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/context"
	"log"
	"time"
)

type asyncExecutor struct {
	requestCh  chan *function.Request
	priorityCh chan *function.Request
	stop       chan bool
}

var AsyncExecutor *asyncExecutor

func InitAsyncExecutor() {
	AsyncExecutor = &asyncExecutor{
		requestCh:  make(chan *function.Request, 500),
		priorityCh: make(chan *function.Request, 500),
		stop:       make(chan bool),
	}

	//start consuming messages for asynchronous invocations
	go AsyncExecutor.consume()
}

//consume messages and executes task
// A message is deleted if and only if it has been consumed correctly (function executed)
//Else it is enqueued to a priority queue. We use two queues because channel don't support the insertAtBeginning feature
// we don't want that after a failure a message has to wait in queue again.
//AsyncExecutor consumes messages from the priorityCh (queue) and if it is empty consumes messages from the requestCh
func (asyncExecutor *asyncExecutor) consume() {
	for {
		select {
		case r := <-asyncExecutor.priorityCh: //used to simulate ghost messages
			asyncExecutor.processRequest(r)
			break
		case r := <-asyncExecutor.requestCh:
			asyncExecutor.processRequest(r)
		case <-asyncExecutor.stop:
			return
		}
	}
}

func (asyncExecutor *asyncExecutor) processRequest(r *function.Request) {
	r.CanDoOffloading = false
	r.Class = function.LOW
	report, err := SubmitRequest(r)
	if err == nil {
		//save report result to etcd
		sendToEtcd(r.AsyncKey, report.Result)
		return
	} else {
		//execution fails, enqueue again the request.
		asyncExecutor.priorityCh <- r
	}
}

func sendToEtcd(key string, result string) {
	etcdClient, err := utils.GetEtcdClient()
	if err != nil {
		log.Fatal("Client not available")
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)

	resp, err := etcdClient.Grant(ctx, 600) //10 minutes
	if err != nil {
		log.Fatal(err)
		return
	}

	_, err = etcdClient.Put(ctx, key, result, clientv3.WithLease(resp.ID))
	if err != nil {
		log.Fatal(err)
		return
	}

	//audit todo delete these lines
	res, err := etcdClient.Get(ctx, key)
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Print(res.Kvs[0].Value)
}
