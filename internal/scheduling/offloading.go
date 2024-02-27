package scheduling

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/metrics"

	"github.com/grussorusso/serverledge/internal/client"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/node"
	"github.com/grussorusso/serverledge/internal/registration"
)

const SCHED_ACTION_OFFLOAD_CLOUD = "O_C"
const SCHED_ACTION_OFFLOAD_EDGE = "O_E"

func pickEdgeNodeForOffloading(r *scheduledRequest) (url string) {
	nearbyServersMap := registration.Reg.NearbyServersMap
	if nearbyServersMap == nil {
		return ""
	}

	//first, search for warm container
	//log.Printf("Search for a warm container")
	for _, v := range nearbyServersMap {
		if v.AvailableWarmContainers[r.Fun.Name] != 0 && v.AvailableCPUs >= r.Request.Fun.CPUDemand {
			return v.Addresses.NodeAddress
		}
	}
	//log.Printf("Nobody has warm container: search for available memory")
	//second, (nobody has warm container) search for available memory
	for _, v := range nearbyServersMap {
		if v.AvailableMemMB >= r.Request.Fun.MemoryMB && v.AvailableCPUs >= r.Request.Fun.CPUDemand {
			return v.Addresses.NodeAddress
		}
	}
	//log.Println("No nearby nodes with enough resources to handle execution.")
	return ""
}

func pickEdgeNodeWithWarmForOffloading(r *scheduledRequest) (url string) {
	nearbyServersMap := registration.Reg.NearbyServersMap
	if nearbyServersMap == nil {
		return ""
	}

	//search for warm container
	for _, v := range nearbyServersMap {
		if v.AvailableWarmContainers[r.Fun.Name] != 0 && v.AvailableCPUs >= r.Request.Fun.CPUDemand {
			return v.Addresses.NodeAddress
		}
	}

	return ""
}

func pickCloudNodeForOffloading() (url string) {
	cloudServersInfoMap, err := registration.Reg.GetAll(true)
	if err != nil {
		return ""
	}

	// search first available node
	for _, values := range cloudServersInfoMap {
		nodeInfo := registration.GetNodeAddresses(values)
		return nodeInfo.NodeAddress
	}

	return ""
}

/* FIXME never used - functions to get warm containers in cloud nodes and to get rtt between edge nodes
func getWarmContainersInCloud(r *scheduledRequest) int {
	cloudServersMap := registration.Reg.CloudServersMap

	sum := 0
	for _, v := range cloudServersMap {
		sum += v.AvailableWarmContainers[r.Fun.Name]
	}

	return sum
}

func getEdgeNodeOffloadingRtt(r *scheduledRequest) (string, float64) {
	nearbyServersMap := registration.Reg.NearbyServersMap
	if nearbyServersMap == nil {
		return "", -1
	}
	//first, search for warm container
	min := math.MaxFloat64
	key := ""

	for k, v := range nearbyServersMap {
		if v.AvailableWarmContainers[r.Fun.Name] != 0 && v.AvailableCPUs >= r.Request.Fun.CPUDemand {
			rtt := float64(registration.Reg.Client.DistanceTo(&v.Coordinates) / time.Millisecond)

			if rtt < min {
				min = rtt
				key = k
			}
		}
	}

	//second, (nobody has warm container) search for available memory
	for k, v := range nearbyServersMap {
		if v.AvailableMemMB >= r.Request.Fun.MemoryMB && v.AvailableCPUs >= r.Request.Fun.CPUDemand {
			rtt := float64(registration.Reg.Client.DistanceTo(&v.Coordinates) / time.Millisecond)

			if rtt < min {
				min = rtt
				key = k
			}
		}
	}

	if key == "" {
		return "", -1
	}
	return nearbyServersMap[key].Url, min / 1000
}*/

func Offload(r *function.Request, serverUrl string) error {
	// Prepare request
	request := client.InvocationRequest{Params: r.Params, QoSClass: r.RequestQoS.ClassService.Name, QoSMaxRespT: r.MaxRespT}
	invocationBody, err := json.Marshal(request)
	if err != nil {
		log.Print(err)
		return err
	}
	sendingTime := time.Now() // used to compute latency later on
	resp, err := offloadingClient.Post(serverUrl+"/invoke/"+r.Fun.Name, "application/json",
		bytes.NewBuffer(invocationBody))

	if err != nil {
		log.Print(err)
		return err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return node.OutOfResourcesErr
		}
		return fmt.Errorf("Remote returned: %v", resp.StatusCode)
	}

	var response function.Response
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if err = json.Unmarshal(body, &response); err != nil {
		return err
	}
	r.ExecReport = response.ExecutionReport
	now := time.Now()
	response.ExecutionReport.ResponseTime = now.Sub(r.Arrival).Seconds()

	if checkIfCloudOffloading(serverUrl) {
		r.ExecReport.OffloadLatencyCloud = time.Now().Sub(sendingTime).Seconds() - r.ExecReport.Duration - r.ExecReport.InitTime
		r.ExecReport.SchedAction = SCHED_ACTION_OFFLOAD_CLOUD
		r.ExecReport.VerticallyOffloaded = true
		r.ExecReport.Cost = config.GetFloat(config.CLOUD_COST_FACTOR, 0.01) * r.ExecReport.Duration * (float64(r.Fun.MemoryMB) / 1024)
		node.Resources.NodeExpenses += r.ExecReport.Cost
		// log.Println("Node total expenses: ", node.Resources.NodeExpenses)
	} else {
		r.ExecReport.OffloadLatencyEdge = time.Now().Sub(sendingTime).Seconds() - r.ExecReport.Duration - r.ExecReport.InitTime
		r.ExecReport.SchedAction = SCHED_ACTION_OFFLOAD_EDGE
		r.ExecReport.VerticallyOffloaded = false
		r.ExecReport.Cost = 0
	}

	policy.OnCompletion(&scheduledRequest{
		Request:         r,
		decisionChannel: nil,
		priority:        0,
	})

	if metrics.Enabled {
		addOffloadedMetrics(r)
	}

	return nil
}

// checkIfCloudOffloading checks if the offloading is vertical or horizontal, given the url of the target server
func checkIfCloudOffloading(serverUrl string) bool {
	allCloudNodes, _ := registration.Reg.GetAll(true)
	for _, value := range allCloudNodes {
		nodeInfo := registration.GetNodeAddresses(value)
		url := nodeInfo.NodeAddress
		if serverUrl == url {
			//log.Printf("Cloud server with key %v was chosen to host offloading", key)
			return true
		} // vertical offloading
	}
	return false // horizontal offloading
}

func addOffloadedMetrics(r *function.Request) {
	metrics.AddCompletedInvocationOffloaded(r.Fun.Name)
	metrics.AddFunctionDurationOffloadedValue(r.Fun.Name, r.ExecReport.Duration)

	if !r.ExecReport.IsWarmStart {
		metrics.AddColdStartOffload(r.Fun.Name, r.ExecReport.InitTime)
	}

	metrics.AddOffloadingTime(r.Fun.Name, r.ExecReport.OffloadLatencyCloud)
	// todo add offloading time for edge
}

func OffloadAsync(r *function.Request, serverUrl string) error {
	// Prepare request
	//QoSClass:    int64(r.Class),
	request := client.InvocationRequest{Params: r.Params,
		QoSMaxRespT: r.MaxRespT,
		Async:       true}
	invocationBody, err := json.Marshal(request)
	if err != nil {
		log.Print(err)
		return err
	}
	resp, err := offloadingClient.Post(serverUrl+"/invoke/"+r.Fun.Name, "application/json",
		bytes.NewBuffer(invocationBody))

	if err != nil {
		log.Print(err)
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Remote returned: %v", resp.StatusCode)
	}

	// there is nothing to wait for
	return nil
}
