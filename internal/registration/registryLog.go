package registration

import (
	"log"
	"reflect"
	"sort"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/hexablock/vivaldi"
)

var Reg *Registry

func InitEdgeMonitoring(r *Registry) (e error) {
	Reg = r
	defaultConfig := vivaldi.DefaultConfig()
	defaultConfig.Dimensionality = 3

	client, err := vivaldi.NewClient(defaultConfig)
	if err != nil {
		log.Fatal(err)
		return err
	}
	Reg.Client = client
	Reg.etcdCh = make(chan bool)
	Reg.serversMap = make(map[string]*StatusInformation)
	Reg.NearbyServersMap = make(map[string]*StatusInformation)

	// start listening for incoming udp connections; use case: edge-nodes request for status infos
	go UDPStatusServer()
	//complete monitoring phase at startup
	monitoring()
	go runMonitor()
	return nil
}

func runMonitor() {
	//todo  adjust default values
	nearbyTicker := time.NewTicker(time.Duration(config.GetInt(config.REG_NEARBY_INTERVAL, 30)) * time.Second)         //wake-up nearby monitoring
	monitoringTicker := time.NewTicker(time.Duration(config.GetInt(config.REG_MONITORING_INTERVAL, 60)) * time.Second) // wake-up general-area monitoring

	for {
		select {
		case <-Reg.etcdCh:
			monitoring()
		case <-monitoringTicker.C:
			monitoring()
		case <-nearbyTicker.C:
			nearbyMonitoring()
		}
	}
}

func monitoring() {
	Reg.RwMtx.Lock()
	defer Reg.RwMtx.Unlock()
	etcdServerMap, err := Reg.GetAll(false)
	if err != nil {
		log.Println(err)
		return
	}

	delete(etcdServerMap, Reg.Key) // not consider myself

	for key, values := range etcdServerMap {
		oldInfo, ok := Reg.serversMap[key]

		nodeInfo := GetNodeAddresses(values)
		url := nodeInfo.RegistryAddress
		hostname := url[7 : len(url)-5]
		port := url[len(url)-4:]
		// use udp socket to retrieve infos about the edge-node status and rtt
		newInfo, rtt := statusInfoRequest(hostname, port)
		if newInfo == nil {
			//unreachable server
			delete(Reg.serversMap, key)
			continue
		}
		Reg.serversMap[key] = newInfo
		if (ok && !reflect.DeepEqual(oldInfo.Coordinates, newInfo.Coordinates)) || !ok {
			Reg.Client.Update("node", &newInfo.Coordinates, rtt)
		}
	}
	//deletes information about servers that haven't registered anymore
	for key := range Reg.serversMap {
		_, ok := etcdServerMap[key]
		if !ok {
			delete(Reg.serversMap, key)
		}
	}

	getRank(2) //todo change this value
}

type dist struct {
	key      string
	distance time.Duration
}

// getRank finds servers nearby to the current one
func getRank(rank int) {
	if rank > len(Reg.serversMap) {
		for k, v := range Reg.serversMap {
			Reg.NearbyServersMap[k] = v
		}
		return
	}

	var distanceBuf = make([]dist, 0) //distances from current server
	for key, s := range Reg.serversMap {
		distanceBuf = append(distanceBuf, dist{key, Reg.Client.DistanceTo(&s.Coordinates)})
	}
	sort.Slice(distanceBuf, func(i, j int) bool { return distanceBuf[i].distance < distanceBuf[j].distance })
	Reg.NearbyServersMap = make(map[string]*StatusInformation)
	for i := 0; i < rank; i++ {
		k := distanceBuf[i].key
		Reg.NearbyServersMap[k] = Reg.serversMap[k]
	}
}

// nearbyMonitoring check nearby server's status
func nearbyMonitoring() {
	Reg.RwMtx.Lock()
	defer Reg.RwMtx.Unlock()

	for key, info := range Reg.NearbyServersMap {
		oldInfo, ok := Reg.serversMap[key]
		regAddress := info.Addresses.RegistryAddress

		hostname := regAddress[7 : len(regAddress)-5]
		port := regAddress[len(regAddress)-4:]
		newInfo, rtt := statusInfoRequest(hostname, port)
		if newInfo == nil {
			//unreachable server
			log.Println("Unreachable server")
			delete(Reg.serversMap, key)
			//trigger a complete monitoring phase
			go func() { Reg.etcdCh <- true }()
			return
		}
		Reg.serversMap[key] = newInfo
		if (ok && !reflect.DeepEqual(oldInfo.Coordinates, newInfo.Coordinates)) || !ok {
			Reg.Client.Update("node", &newInfo.Coordinates, rtt)
		}
	}
}
