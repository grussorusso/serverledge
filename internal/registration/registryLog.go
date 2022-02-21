package registration

import (
	"encoding/json"
	"fmt"
	"github.com/grussorusso/serverledge/internal/config"
	"github.com/hexablock/vivaldi"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"sort"
	"time"
)

var Reg *Registry

func Init(r *Registry) (e error) {
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
	serversMap = make(map[string]*StatusInformation)
	nearbyServersMap = make(map[string]*StatusInformation)
	go runMonitor()
	return nil
}

func runMonitor() {
	//todo  adjust default values
	nearbyTicker := time.NewTicker(time.Duration(config.GetInt("registry.nearby.interval", 15)) * time.Second)         //wake-up nearby monitoring
	monitoringTicker := time.NewTicker(time.Duration(config.GetInt("registry.monitoring.interval", 30)) * time.Second) // wake-up general-area monitoring
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
	etcdServerMap, err := Reg.GetAll()
	if err != nil {
		log.Println(err)
		return
	}

	delete(etcdServerMap, Reg.Key) // not consider myself
	for key, url := range etcdServerMap {
		oldInfo, ok := serversMap[key]
		newInfo, rtt := getStatusInformation(url)
		if newInfo == nil {
			//unreachable server
			delete(serversMap, key)
			continue
		}
		serversMap[key] = newInfo
		if (ok && !reflect.DeepEqual(oldInfo.Coordinates, newInfo.Coordinates)) || !ok {
			Reg.Client.Update("node", &newInfo.Coordinates, rtt)
		}
	}
	//deletes information about servers that haven't registered anymore
	for key := range serversMap {
		_, ok := etcdServerMap[key]
		if !ok {
			delete(serversMap, key)
		}
	}

	getRank(2) //todo change this value
}

func getStatusInformation(url string) (info *StatusInformation, duration time.Duration) {
	initTime := time.Now()
	resp, err := http.Get(url + "status")
	rtt := time.Now().Sub(initTime)
	if err != nil {
		fmt.Printf("Invocation failed: %v", err)
		return nil, 0
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body) // response body is []byte
	if err != nil {
		fmt.Printf("ReadAll failed: %v", err)
		return nil, 0
	}

	var result StatusInformation
	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println("Can not unmarshal JSON")
		return nil, 0
	}

	return &result, rtt
}

type dist struct {
	key      string
	distance time.Duration
}

//getRank finds servers nearby to the current one
func getRank(rank int) {
	if rank > len(serversMap) {
		for k, v := range serversMap {
			nearbyServersMap[k] = v
		}
		return
	}

	var distanceBuf = make([]dist, 0) //distances from current server
	for key, s := range serversMap {
		distanceBuf = append(distanceBuf, dist{key, Reg.Client.DistanceTo(&s.Coordinates)})
	}
	sort.Slice(distanceBuf, func(i, j int) bool { return distanceBuf[i].distance < distanceBuf[j].distance })
	nearbyServersMap = make(map[string]*StatusInformation)
	for i := 0; i < rank; i++ {
		k := distanceBuf[i].key
		nearbyServersMap[k] = serversMap[k]
	}
}

// nearbyMonitoring check nearby server's status
func nearbyMonitoring() {
	for key, info := range nearbyServersMap {
		oldInfo, ok := serversMap[key]
		newInfo, rtt := getStatusInformation(info.Url)
		if newInfo == nil {
			//unreachable server
			delete(serversMap, key)
			//trigger a complete monitoring phase
			go func() { Reg.etcdCh <- true }()
			return
		}
		serversMap[key] = newInfo
		if (ok && !reflect.DeepEqual(oldInfo.Coordinates, newInfo.Coordinates)) || !ok {
			Reg.Client.Update("node", &newInfo.Coordinates, rtt)
		}
	}
}
