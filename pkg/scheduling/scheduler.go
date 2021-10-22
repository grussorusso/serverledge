package scheduling

import "github.com/grussorusso/serverledge/pkg/containers"
import "github.com/grussorusso/serverledge/pkg/functions"

func Schedule (r *functions.Request) (string, error) {
	containerID, warm := containers.GetWarmContainer(r.Fun)
	if warm {
		return containers.WarmStart(r, containerID)
	} else {
		return containers.ColdStart(r)
	}
}

