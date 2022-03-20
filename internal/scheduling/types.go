package scheduling

import (
	"github.com/grussorusso/serverledge/internal/container"
	"github.com/grussorusso/serverledge/internal/function"
)

// scheduledRequest represents a Request within the scheduling subsystem
type scheduledRequest struct {
	*function.Request
	decisionChannel chan schedDecision
}

// schedDecision wraps a action made by the scheduler.
// Possible decisions are 1) drop, 2) execute locally or 3) execute on a remote
// Node (offloading).
type schedDecision struct {
	action     action
	contID     container.ContainerID
	remoteHost string
}

type action int64

const (
	DROP         action = 0
	EXEC_LOCAL          = 1
	EXEC_REMOTE         = 2
	RETURN_ASYNC        = 3
)
