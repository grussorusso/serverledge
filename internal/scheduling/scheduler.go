package scheduling

import (
	"fmt"
	"log"
	"time"

	"github.com/grussorusso/serverledge/internal/containers"
	"github.com/grussorusso/serverledge/internal/functions"
)

func Schedule(r *functions.Request) (*functions.ExecutionReport, error) {
	schedArrivalT := time.Now()
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

	initTime := time.Now().Sub(schedArrivalT).Seconds()
	r.Report = &functions.ExecutionReport{InitTime: initTime}

	return containers.Invoke(containerID, r)
}
