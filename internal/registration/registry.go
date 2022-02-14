package registration

import (
	"fmt"
	"github.com/grussorusso/serverledge/utils"
	"github.com/lithammer/shortuuid"
	_ "go.etcd.io/etcd/client/v3"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/context"
	"log"
	"strconv"
	"time"
)

var BASEDIR = "registry"

type Registry struct {
	Area string
	id   string
}

// getEtcdKey append to a given unique id the logical path depending on the Area.
// If it is called with  an empty string  it returns the base path for the current local Area.
func (r *Registry) getEtcdKey(id string) (key string) {
	return fmt.Sprintf("%s/%s/%s", BASEDIR, r.Area, id)
}

// RegisterToEtcd make a registration to the local Area; etcd put operation is performed
func (r *Registry) RegisterToEtcd(url string) (e error) {
	etcdClient, err := utils.GetEtcdClient()
	if err != nil {
		log.Fatal("etcd client unavailable.")
		return err
	}

	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	//generate unique identifier
	id := shortuuid.New() + strconv.FormatInt(time.Now().UnixNano(), 10)
	r.id = id
	// save couple (id, url) to the correct Area-dir on etcd
	_, err = etcdClient.Put(ctx, r.getEtcdKey(r.id), url)
	if err != nil {
		log.Fatal("etcd error: could not complete the registration.")
		return err
	}

	return nil
}

//GetAll is used to obtain the list of  other server's addresses under a specific local Area
func (r *Registry) GetAll() ([]string, error) {
	baseDir := r.getEtcdKey("")
	ctx := context.TODO()
	etcdClient, err := utils.GetEtcdClient()
	if err != nil {
		log.Fatal("etcd client unavailable.")
		return nil, err
	}
	//retrieve all url of the other servers under my Area
	resp, err := etcdClient.Get(ctx, baseDir, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	servers := make([]string, len(resp.Kvs))
	for i, s := range resp.Kvs {
		servers[i] = string(s.Value)
		//audit todo delete the next line
		log.Printf("found edge server at: %s", servers[i])
	}

	return servers, nil
}

// Deregister deletes from etcd the key, value pair previously inserted
func (r *Registry) Deregister() (e error) {
	etcdClient, err := utils.GetEtcdClient()
	if err != nil {
		log.Fatal("etcd client unavailable.")
		return err
	}

	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	_, err = etcdClient.Delete(ctx, r.getEtcdKey(r.id))
	if err != nil {
		return err
	}

	log.Println("Deregister : " + r.id)
	return nil
}
