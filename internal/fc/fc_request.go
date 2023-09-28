package fc

import (
	"github.com/grussorusso/serverledge/internal/function"
	"time"
)

// CompositionRequest represents a single function composition internal invocation, with params and metrics data
type CompositionRequest struct {
	ReqId           string
	Fc              *FunctionComposition
	Params          map[string]interface{}
	Arrival         time.Time
	ExecReport      CompositionExecutionReport     // each function has its execution report, and the composition has additional metrics
	RequestQoSMap   map[string]function.RequestQoS // every function should have its RequestQoS
	CanDoOffloading bool                           // every function inherits this flag
	Async           bool
}

func NewCompositionRequest(reqId string, composition *FunctionComposition, params map[string]interface{}) *CompositionRequest {
	return &CompositionRequest{
		ReqId:           reqId,
		Fc:              composition,
		Params:          params,
		Arrival:         time.Now(),
		ExecReport:      CompositionExecutionReport{},
		RequestQoSMap:   make(map[string]function.RequestQoS),
		CanDoOffloading: true,
		Async:           false,
	}
}

type CompositionResponse struct {
	Success bool
	CompositionExecutionReport
}

type CompositionAsyncResponse struct {
	ReqId string
}
