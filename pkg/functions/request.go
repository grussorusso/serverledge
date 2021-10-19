package functions

import "fmt"
import "time"

type Request struct {
	Fun *Function
	Arrival time.Time
}

func (r *Request) String() string {
	return fmt.Sprintf("Req-%s", r.Fun.Name)
}

