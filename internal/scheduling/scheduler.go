package scheduling

import (
	"fmt"
	"log"

	"github.com/grussorusso/serverledge/internal/containers"
	"github.com/grussorusso/serverledge/internal/functions"
)

func Schedule(r *functions.Request) (*functions.ExecutionReport, error) {
	containerID, ok := containers.AcquireWarmContainer(r.Fun)
	if !ok {
		newContainer, err := containers.NewContainer(r.Fun)
		if err != nil {
			// TODO: this may fail because we run out of memory/CPU
			// handle this error differently?
			return nil, fmt.Errorf("Could not create a new container: %v", err)
		}
		containerID = newContainer
	} else {
		log.Printf("Using a warm container for: %v", r)
	}

	// TODO: defer marking the container as ready

	result, err := containers.Invoke(containerID, r)
	if err != nil {
		report := &functions.ExecutionReport{Success: err != nil, Output: result}
		return report, nil
	} else {
		return nil, err
	}
}
