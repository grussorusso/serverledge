package scheduling

import "github.com/grussorusso/serverledge/pkg/containers"
import "github.com/grussorusso/serverledge/pkg/functions"

func Schedule (r *functions.Request) error {
	containerID, warm := containers.GetWarmContainer(r.Fun)
	if warm {
		return containers.WarmStart(r, containerID)
	} else {
		go containers.ColdStart(r)
	}
	return nil
}

