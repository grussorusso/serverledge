package utils

import (
	"fmt"
	"sync"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var etcdClient *clientv3.Client = nil
var clientMutex sync.Mutex

func GetEtcdClient() (*clientv3.Client, error) {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	// reuse client
	if etcdClient != nil {
		return etcdClient, nil
	}

	etcdHost := config.GetString(config.ETCD_ADDRESS, "localhost:2379")
	timeout := config.GetInt(config.ETCD_TIMEOUT, 1)
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdHost},
		DialTimeout: time.Duration(timeout) * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("Could not connect to etcd: %v", err)
	}

	etcdClient = cli
	return cli, nil
}
