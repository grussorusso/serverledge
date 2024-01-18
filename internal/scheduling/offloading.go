package scheduling

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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

func Offload(r *function.Request, serverUrl string) error {
	// Prepare request
	request := client.InvocationRequest{Params: r.Params, QoSClass: r.Class, QoSMaxRespT: r.MaxRespT}
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
	// It was originially computed as "report.Arrival - sendingTime"
	r.ExecReport.OffloadLatency = time.Now().Sub(sendingTime).Seconds() - r.ExecReport.Duration - r.ExecReport.InitTime
	r.ExecReport.SchedAction = SCHED_ACTION_OFFLOAD

	return nil
}

func OffloadAsync(r *function.Request, serverUrl string) error {
	// Prepare request
	request := client.InvocationRequest{Params: r.Params,
		QoSClass:    r.Class,
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
