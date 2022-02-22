package scheduling

import (
	"errors"
	"log"
)

type Policy interface {
	Init()
	OnCompletion(request *scheduledRequest)
	OnArrival(request *scheduledRequest)
}

type DefaultLocalPolicy struct{}

func (p *DefaultLocalPolicy) Init() {

}

func (p *DefaultLocalPolicy) OnCompletion(r *scheduledRequest) {

}

func (p *DefaultLocalPolicy) OnArrival(r *scheduledRequest) {
	containerID, err := acquireWarmContainer(r.Fun)
	if err == nil {
		log.Printf("Using a warm container for: %v", r)
		execLocally(r, containerID, true)
	} else if errors.Is(err, NoWarmFoundErr) {
		// Cold Start (handles asynchronously)
		go handleColdStart(r)
	} else if errors.Is(err, OutOfResourcesErr) {
		log.Printf("Not enough resources on the Node.")
		dropRequest(r)
	} else {
		// other error
		dropRequest(r)
	}
}
