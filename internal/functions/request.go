package functions

import (
	"fmt"
	"time"
)

//Request represents a single function invocation.
type Request struct {
	Fun     *Function
	Params  map[string]string
	Arrival time.Time
}

func (r *Request) String() string {
	return fmt.Sprintf("Req-%s", r.Fun.Name)
}
