package fc

import (
	"github.com/grussorusso/serverledge/internal/function"
	"time"
)

// Request represents a single function composition invocation.
type Request struct {
	ReqId      string
	Fc         *FunctionComposition
	Params     map[string]interface{}
	Arrival    time.Time
	ExecReport function.ExecutionReport
	function.RequestQoS
	CanDoOffloading bool
	Async           bool
}
