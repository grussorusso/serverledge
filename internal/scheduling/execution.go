package scheduling

import (
	"fmt"
	"github.com/grussorusso/serverledge/internal/resources_mgnt"
	"log"
	"time"

	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/executor"
	"github.com/grussorusso/serverledge/internal/function"
)

const HANDLER_DIR = "/app"

// Execute serves a request on the specified container.
func Execute(contID container.ContainerID, r *scheduledRequest) (*function.ExecutionReport, error) {
	defer resources_mgnt.ReleaseContainer(contID, r.Fun)

	log.Printf("Invoking function on container: %v", contID)

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

	response, err := container.Execute(contID, &req)
	if err != nil {
		return nil, fmt.Errorf("Execution request failed: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("Function execution failed")
	}

	r.Report.Result = response.Result
	r.Report.Duration = response.Duration
	r.Report.ResponseTime = time.Now().Sub(r.Arrival).Seconds()
	r.Report.CPUTime = response.CPUTime

	// notify scheduler
	completions <- r

	return r.Report, nil
}
