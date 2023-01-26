package utils

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var etcdClient *clientv3.Client = nil
var clientMutex sync.Mutex
var Timeout time.Duration

func GetEtcdClient() (*clientv3.Client, error) {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	// reuse client
	if etcdClient != nil {
		return etcdClient, nil
	}

	Timeout = time.Duration(config.GetInt(config.ETCD_TIMEOUT, 1)) * time.Second
	log.Println("Dial Timeout for etcd client: ", Timeout)
	etcdHost := config.GetString(config.ETCD_ADDRESS, "localhost:2379")

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdHost},
		DialTimeout: Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("Could not connect to etcd: %v", err)
	}

	etcdClient = cli
	return cli, nil
}
