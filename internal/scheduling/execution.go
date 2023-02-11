package scheduling

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/executor"
	"github.com/grussorusso/serverledge/internal/node"
)

const HANDLER_DIR = "/app"

// Execute serves a request on the specified container.
func Execute(contID container.ContainerID, r *scheduledRequest) error {
	//log.Printf("[%s] Executing on container: %v", r, contID)

	//Retrieve the local IP address of the node that will execute the request
	hostName, _ := os.Hostname()
	addresses, _ := net.LookupHost(hostName)
	localIp := addresses[0]

	var req executor.InvocationRequest
	if r.Fun.Runtime == container.CUSTOM_RUNTIME {
		req = executor.InvocationRequest{
			Params: r.Params,
		}
	} else {
		cmd := container.RuntimeToInfo[r.Fun.Runtime].InvocationCmd
		req = executor.InvocationRequest{
			Id:              r.ReqId,
			Command:         cmd,
			Params:          r.Params,
			Handler:         r.Fun.Handler,
			HandlerDir:      HANDLER_DIR,
			NodeIP:          localIp,
			Async:           r.Async,
			OriginalRequest: *r.Request,
		}
	}
	node.Resources.Lock()
	node.NodeRequests[contID] = req // Add this request to the pool
	node.Resources.Unlock()

	t0 := time.Now()

	response, invocationWait, err := container.Execute(contID, &req)
	if err != nil {
		// notify scheduler
		completions <- &completion{scheduledRequest: r, contID: contID}
		node.Resources.Lock()
		delete(node.NodeRequests, contID)
		node.Resources.Unlock()
		return fmt.Errorf("[%s] Execution failed: %v", r, err)
	}

	if !response.Success {
		// notify scheduler
		completions <- &completion{scheduledRequest: r, contID: contID}
		node.Resources.Lock()
		delete(node.NodeRequests, contID)
		node.Resources.Unlock()
		return fmt.Errorf("Function execution failed")
	}

	r.ExecReport.Result = response.Result
	r.ExecReport.Duration = time.Now().Sub(t0).Seconds() - invocationWait.Seconds()
	r.ExecReport.ResponseTime = time.Now().Sub(r.Arrival).Seconds()
	r.ExecReport.CPUTime = -1.0 // TODO

	// initializing containers may require invocation retries, adding
	// latency
	r.ExecReport.InitTime += invocationWait.Seconds()

	// notify scheduler
	completions <- &completion{scheduledRequest: r, contID: contID}
	node.Resources.Lock()
	delete(node.NodeRequests, contID)
	node.Resources.Unlock()
	return nil
}

func Checkpoint(contID container.ContainerID, fallbackAddresses []string) error {
	req := executor.FallbackAcquisitionRequest{
		FallbackAddresses: fallbackAddresses,
	}
	response, checkpointTime, err := container.Checkpoint(contID, &req)
	if err != nil || !response.Success {
		// notify scheduler
		return fmt.Errorf("Checkpoint failed: %v", err)
	}
	fmt.Println("Checkpoint succeded in time ", checkpointTime)
	file, err := os.OpenFile("checkpointlog.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	file.WriteString(checkpointTime.String() + "\n")
	return nil
}

func Restore(contID container.ContainerID, archiveName string) error {

	restoreTime, err := container.Restore(contID, archiveName)
	if err != nil {
		// notify scheduler
		return fmt.Errorf("Restore failed: %v", err)
	}
	fmt.Println("Restore succeded in time ", restoreTime)
	file, err := os.OpenFile("restorelog.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	file.WriteString(restoreTime.String() + "\n")
	return nil
}
