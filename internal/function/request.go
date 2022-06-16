package function

import (
	"fmt"
	"time"

	"github.com/grussorusso/serverledge/internal/config"
)

//Request represents a single function invocation.
type Request struct {
	Fun     *Function
	Params  map[string]interface{}
	Arrival time.Time
	Report  *ExecutionReport
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

// MaxRespTime todo adjust response time -> Second default unit
var MaxRespTime = config.GetFloat("max.response.time", 20) //in Seconds

type ServiceClass int64

const (
	LOW               ServiceClass = 0
	HIGH_PERFORMANCE               = 1
	HIGH_AVAILABILITY              = 2
)
