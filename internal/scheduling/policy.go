package scheduling

import (
	"errors"
	"github.com/grussorusso/serverledge/internal/node"
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
	containerID, err := node.AcquireWarmContainer(r.Fun)
	if err == nil {
		log.Printf("Using a warm container for: %v", r)
		execLocally(r, containerID, true)
	} else if errors.Is(err, node.NoWarmFoundErr) {
		// Cold Start (handles asynchronously)
		go func() {
			if !handleColdStart(r) {
				dropRequest(r)
			}
		}()
	} else if errors.Is(err, node.OutOfResourcesErr) {
		log.Printf("Not enough resources on the Resources.")
		dropRequest(r)
	} else {
		// other error
		dropRequest(r)
	}
}
