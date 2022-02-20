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

var registry Registry

//Init todo call this in main
func Init(r Registry) (e error) {
	registry = r
	defaultConfig := vivaldi.DefaultConfig()
	defaultConfig.Dimensionality = 3

	client, err := vivaldi.NewClient(defaultConfig)
	if err != nil {
		log.Fatal(err)
		return err
	}
	registry.client = client
	go runMonitor()
	registry.etcdCh <- true //trigger initialization
	return nil
}

func runMonitor() {
	nearbyTicker := time.NewTicker(time.Duration(config.GetInt("registry.nearby.interval", 60)) * time.Second)
	monitoringTicker := time.NewTicker(time.Duration(config.GetInt("registry.monitoring.interval", 5)) * time.Minute)
	for {
		select {
		case <-registry.etcdCh:
			monitoring()
		case <-monitoringTicker.C:
			monitoring()
		case <-nearbyTicker.C:
			nearbyMonitoring()
		}
	}
}

func monitoring() {
	etcdServerMap, err := registry.GetAll()
	if err != nil {
		log.Println(err)
		return
	}

	delete(etcdServerMap, registry.Key) // not consider myself
	for key, url := range etcdServerMap {
		oldInfo, ok := serversMap[key]
		newInfo, rtt := getStatusInformation(url)
		if newInfo == nil {
			//unreachable server
			delete(serversMap, key)
			continue
		}
		serversMap[key] = newInfo
		if ok && !reflect.DeepEqual(oldInfo.Coordinates, newInfo.Coordinates) {
			registry.client.Update(oldInfo.Url, &newInfo.Coordinates, rtt)
		}
	}
	//deletes information about servers that haven't registered anymore
	for key := range serversMap {
		_, ok := etcdServerMap[key]
		if !ok {
			delete(serversMap, key)
		}
	}

	getRank(3) //todo change this value
}

func getStatusInformation(url string) (info *StatusInformation, duration time.Duration) {
	initTime := time.Now()
	resp, err := http.Get(url)
	rtt := time.Now().Sub(initTime)
	if err != nil {
		fmt.Printf("Invocation failed: %v", err)
		return nil, 0
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body) // response body is []byte

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
	var distanceBuf = make([]dist, len(serversMap)) //distances from current server
	for key, s := range serversMap {
		distanceBuf = append(distanceBuf, dist{key, registry.client.DistanceTo(&s.Coordinates)})
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
			go func() { registry.etcdCh <- true }()
			return
		}
		serversMap[key] = newInfo
		if ok && !reflect.DeepEqual(oldInfo.Coordinates, newInfo.Coordinates) {
			registry.client.Update(oldInfo.Url, &newInfo.Coordinates, rtt)
		}
	}
}
