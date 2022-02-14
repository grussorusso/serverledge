package function

import (
	"fmt"
	"time"
)

//Request represents a single function invocation.
type Request struct {
	Fun     *Function
	Params  map[string]string
	Arrival time.Time
	Report  *ExecutionReport
	RequestQoS
}

type RequestQoS struct {
	Class    string
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
}

func (r *Request) String() string {
	return fmt.Sprintf("Req-%s", r.Fun.Name)
}
