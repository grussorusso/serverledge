package scheduling

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/grussorusso/serverledge/internal/metrics"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/grussorusso/serverledge/internal/client"
	"github.com/grussorusso/serverledge/internal/function"
	"github.com/grussorusso/serverledge/internal/registration"
)

const SCHED_ACTION_OFFLOAD = "O"

func pickEdgeNodeForOffloading(r *scheduledRequest) (url string) {
	nearbyServersMap := registration.Reg.NearbyServersMap
	if nearbyServersMap == nil {
		return ""
	}
	//first, search for warm container
	for _, v := range nearbyServersMap {
		if v.AvailableWarmContainers[r.Fun.Name] != 0 && v.AvailableCPUs >= r.Request.Fun.CPUDemand {
			return v.Url
		}
	}
	//second, (nobody has warm container) search for available memory
	for _, v := range nearbyServersMap {
		if v.AvailableMemMB >= r.Request.Fun.MemoryMB && v.AvailableCPUs >= r.Request.Fun.CPUDemand {
			return v.Url
		}
	}
	return ""
}

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
}

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
		return fmt.Errorf("Remote returned: %v", resp.StatusCode)
	}

	var response function.Response
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	if err = json.Unmarshal(body, &response); err != nil {
		return err
	}
	r.ExecReport = response.ExecutionReport

	// TODO: check how this is used in the QoSAware policy
	// It was originally computed as "report.Arrival - sendingTime"
	// Check if r.ExecReport.Duration and r.ExecReport.InitTime are greater than 0
	log.Printf("OFFLOADING RESULT Duration %f, InitTime: %f", r.ExecReport.Duration, r.ExecReport.InitTime)

	r.ExecReport.OffloadLatency = time.Now().Sub(sendingTime).Seconds() - r.ExecReport.Duration - r.ExecReport.InitTime
	r.ExecReport.SchedAction = SCHED_ACTION_OFFLOAD

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

func addOffloadedMetrics(r *function.Request) {
	metrics.AddCompletedInvocationOffloaded(r.Fun.Name)
	metrics.AddFunctionDurationOffloadedValue(r.Fun.Name, r.ExecReport.Duration)

	if !r.ExecReport.IsWarmStart {
		metrics.AddColdStartOffload(r.Fun.Name, r.ExecReport.InitTime)
	}

	metrics.AddOffloadingTime(r.Fun.Name, r.ExecReport.OffloadLatency)
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
