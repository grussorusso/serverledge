package scheduling

import "github.com/grussorusso/serverledge/internal/containers"

// SchedDecision wraps a decision made by the scheduler.
// Possible decisions are 1) drop, 2) execute locally or 3) execute on a remote
// node (offloading).
type SchedDecision struct {
	Decision   Action
	ContID     containers.ContainerID
	RemoteHost string
}

type Action int64

const (
	DROP        Action = 0
	EXEC_LOCAL         = 1
	EXEC_REMOTE        = 2
)
