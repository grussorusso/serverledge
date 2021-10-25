package scheduling

import (
	"fmt"

	"github.com/grussorusso/serverledge/internal/containers"
	"github.com/grussorusso/serverledge/internal/functions"
)

func Schedule(r *functions.Request) (string, error) {
	containerID, ok := containers.GetWarmContainer(r.Fun)
	if !ok {
		newContainer, err := containers.NewContainer(r.Fun)
		if err != nil {
			// TODO: this may fail because we run out of memory/CPU
			// handle this error differently?
			return "", fmt.Errorf("Could not create a new container: %v", err)
		}
		containerID = newContainer
	}

	return containers.Invoke(containerID, r)
}
