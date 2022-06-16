package function

import (
	"fmt"
	"time"
)

//Request represents a single function invocation.
type Request struct {
	Fun        *Function
	Params     map[string]interface{}
	Arrival    time.Time
	ExecReport ExecutionReport
	RequestQoS
	CanDoOffloading bool
}

type RequestQoS struct {
	Class    ServiceClass
	MaxRespT float64
}

type ExecutionReport struct {
	Result         string
	ResponseTime   float64
	IsWarmStart    bool
	InitTime       float64
	OffloadLatency float64
	Duration       float64
	CPUTime        float64
	SchedAction    string
}

func (r *Request) String() string {
	return fmt.Sprintf("Rq-%s-%d", r.Fun.Name, r.Arrival.UnixNano())
}

type ServiceClass int64

const (
	LOW               ServiceClass = 0
	HIGH_PERFORMANCE               = 1
	HIGH_AVAILABILITY              = 2
)
