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

// getEtcdKey append to a given unique id the logical path depending on the Area.
// If it is called with  an empty string  it returns the base path for the current local Area.
func (r *Registry) getEtcdKey(id string) (key string) {
	return fmt.Sprintf("%s/%s/%s", BASEDIR, r.Area, id)
}

// RegisterToEtcd make a registration to the local Area; etcd put operation is performed
func (r *Registry) RegisterToEtcd(url string) (e error) {
	etcdClient, err := utils.GetEtcdClient()
	if err != nil {
		log.Fatal(UnavailableClientErr)
		return UnavailableClientErr
	}

	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	//generate unique identifier
	id := shortuuid.New() + strconv.FormatInt(time.Now().UnixNano(), 10)
	r.id = id

	resp, err := etcdClient.Grant(ctx, int64(TTL))
	if err != nil {
		log.Fatal(err)
		return err
	}

	log.Printf("Registration key: %s\n", r.getEtcdKey(r.id))
	// save couple (id, url) to the correct Area-dir on etcd
	_, err = etcdClient.Put(ctx, r.getEtcdKey(r.id), url, clientv3.WithLease(resp.ID))
	if err != nil {
		log.Fatal(IdRegistrationErr)
		return IdRegistrationErr
	}

	cancelCtx, _ := context.WithCancel(etcdClient.Ctx())

	// the key id will be kept alive until a fault will occur
	keepAliveCh, err := etcdClient.KeepAlive(cancelCtx, resp.ID)
	if err != nil || keepAliveCh == nil {
		log.Fatal(KeepAliveErr)
		return KeepAliveErr
	}

	go func() {
		for alive := range keepAliveCh {
			// eat messages until keep alive channel closes
			log.Println(alive.ID)
		}
	}()

	return nil
}

//GetAll is used to obtain the list of  other server's addresses under a specific local Area
func (r *Registry) GetAll() ([]ServerInformation, error) {
	baseDir := r.getEtcdKey("")
	ctx := context.TODO()
	etcdClient, err := utils.GetEtcdClient()
	if err != nil {
		log.Fatal(UnavailableClientErr)
		return nil, UnavailableClientErr
	}
	//retrieve all url of the other servers under my Area
	resp, err := etcdClient.Get(ctx, baseDir, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	servers := make([]ServerInformation, len(resp.Kvs))
	for i, s := range resp.Kvs {
		servers[i].ipv4 = string(s.Value)
		servers[i].id = string(s.Key)
		//audit todo delete the next line
		log.Printf("found edge server at: %s", servers[i])
	}

	return servers, nil
}

// Deregister deletes from etcd the key, value pair previously inserted
func (r *Registry) Deregister() (e error) {
	etcdClient, err := utils.GetEtcdClient()
	if err != nil {
		log.Fatal(UnavailableClientErr)
		return UnavailableClientErr
	}

	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	_, err = etcdClient.Delete(ctx, r.getEtcdKey(r.id))
	if err != nil {
		return err
	}

	log.Println("Deregister : " + r.id)
	return nil
}
