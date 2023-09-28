package registration

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/LK4D4/trylock"
	"github.com/hexablock/vivaldi"
)

var UnavailableClientErr = errors.New("etcd client unavailable")
var IdRegistrationErr = errors.New("etcd error: could not complete the registration")
var KeepAliveErr = errors.New(" The system can't renew your registration key")

type Registry struct {
	Area             string
	Key              string
	Client           *vivaldi.Client
	RwMtx            trylock.Mutex
	NearbyServersMap map[string]*StatusInformation
	serversMap       map[string]*StatusInformation
	etcdCh           chan bool
}

type StatusInformation struct {
	Addresses               NodeInterfaces
	AvailableWarmContainers map[string]int // <k, v> = <function name, warm container number>
	AvailableMemMB          int64
	AvailableCPUs           float64
	DropCount               int64
	RequestCount            int64
	Coordinates             vivaldi.Coordinate
}

type NodeInterfaces struct {
	NodeAddress     string
	RegistryAddress string
}

// GetNodeAddresses writes the information present into the etcd entry of a node into a struct NodeInterfaces
func GetNodeAddresses(etcdValue string) *NodeInterfaces {
	var nodeInfo NodeInterfaces

	// Get registry address of the target node registry server
	err := json.Unmarshal([]byte(etcdValue), &nodeInfo)
	if err != nil {
		log.Println("Cannot unmarshal target node info recovered from etcd.")
		return nil
	}
	return &nodeInfo
}
