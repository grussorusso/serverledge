package scheduling

import (
	"fmt"
	"log"
	"time"

	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/executor"
	"github.com/grussorusso/serverledge/internal/function"
)

const HANDLER_DIR = "/app"

// Execute serves a request on the specified container.
func Execute(contID container.ContainerID, r *scheduledRequest) (*function.ExecutionReport, error) {
	//defer node.ReleaseContainer(contID, r.Fun)

	log.Printf("[%s] Executing on container: %v", r, contID)

	var req executor.InvocationRequest
	if r.Fun.Runtime == container.CUSTOM_RUNTIME {
		req = executor.InvocationRequest{
			Params: r.Params,
		}
	} else {
		cmd := container.RuntimeToInfo[r.Fun.Runtime].InvocationCmd
		req = executor.InvocationRequest{
			cmd,
			r.Params,
			r.Fun.Handler,
			HANDLER_DIR,
		}
	}

	t0 := time.Now()

	response, err := container.Execute(contID, &req)
	if err != nil {
		return nil, fmt.Errorf("[%s] Execution failed: %v", r, err)
	}

	if !response.Success {
		return nil, fmt.Errorf("Function execution failed")
	}

	r.Report.Result = response.Result
	r.Report.Duration = time.Now().Sub(t0).Seconds()
	r.Report.ResponseTime = time.Now().Sub(r.Arrival).Seconds()
	r.Report.CPUTime = -1.0 // TODO

	// notify scheduler
	completions <- &completion{scheduledRequest: r, contID: contID}

	return r.Report, nil
}
