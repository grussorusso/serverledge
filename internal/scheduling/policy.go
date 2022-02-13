package scheduling

import (
	"errors"
	"log"
)

type Policy interface {
	OnCompletion(request *scheduledRequest)
	OnArrival(request *scheduledRequest)
}

type defaultLocalPolicy struct{}

func (p *defaultLocalPolicy) OnCompletion(r *scheduledRequest) {

}

func (p *defaultLocalPolicy) OnArrival(r *scheduledRequest) {
	containerID, err := acquireWarmContainer(r.Fun)
	if err == nil {
		log.Printf("Using a warm container for: %v", r)
		execLocally(r, containerID)
	} else if errors.Is(err, NoWarmFoundErr) {
		// Cold Start (handles asynchronously)
		go handleColdStart(r)
	} else if errors.Is(err, OutOfResourcesErr) {
		log.Printf("Not enough resources on the node.")
		dropRequest(r)
	} else {
		// other error
		dropRequest(r)
	}
}
