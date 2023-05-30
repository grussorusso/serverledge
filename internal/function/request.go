package function

import (
	"fmt"
	"time"
)

// Request represents a single function invocation.
type Request struct {
	ReqId      string
	Fun        *Function
	Params     map[string]interface{}
	Arrival    time.Time
	ExecReport ExecutionReport
	RequestQoS
	CanDoOffloading bool
	Async           bool
}

type RequestQoS struct {
	Class        ServiceClass
	ClassService QoSClass
	MaxRespT     float64
}

type ExecutionReport struct {
	Name           string
	Class          string
	Result         string
	ResponseTime   float64
	IsWarmStart    bool
	InitTime       float64
	OffloadLatency float64
	Duration       float64
	SchedAction    string
}

type Response struct {
	Success bool
	ExecutionReport
}

type AsyncResponse struct {
	ReqId string
}

func (r *Request) String() string {
	return fmt.Sprintf("Rq-%s", r.Fun.Name, r.ReqId)
}

/*
type ServiceClass struct {
	name                string
	utility             float64
	maximumResponseTime float64
	completedPercentage float64
}
*/

type QoSClass struct {
	Name                string
	Utility             float64
	MaximumResponseTime float64 `default:"-1"`
	CompletedPercentage float64 `default:"0"`
}

func (r Request) GetMaxRT() float64 {
	//Permit lower response time?

	//
	if r.RequestQoS.ClassService.MaximumResponseTime < 0 {
		return r.RequestQoS.MaxRespT
	}

	if r.RequestQoS.MaxRespT > r.RequestQoS.ClassService.MaximumResponseTime {
		return r.RequestQoS.MaxRespT
	} else {
		return r.RequestQoS.ClassService.MaximumResponseTime
	}
}

type ServiceClass int64

const (
	LOW               ServiceClass = 0
	HIGH_PERFORMANCE               = 1
	HIGH_AVAILABILITY              = 2
)
