package scheduling

import "github.com/grussorusso/serverledge/internal/containers"
import "github.com/grussorusso/serverledge/internal/functions"

func Schedule (r *functions.Request) (string, error) {
	// TODO: refactor: get containerID and then invoke on container
	containerID, warm := containers.GetWarmContainer(r.Fun)
	if warm {
		return containers.WarmStart(r, containerID)
	} else {
		return containers.ColdStart(r)
	}
}

