package scheduling

import (
	"errors"
	"log"

	"github.com/grussorusso/serverledge/internal/config"
	"github.com/grussorusso/serverledge/internal/node"
)

type DefaultLocalPolicy struct {
	queue queue
}

func (p *DefaultLocalPolicy) Init() {
	queueCapacity := config.GetInt(config.SCHEDULER_QUEUE_CAPACITY, 0)
	p.queue = NewFIFOQueue(queueCapacity)
}

func (p *DefaultLocalPolicy) OnCompletion(r *scheduledRequest) {
	if p.queue == nil || p.queue.Len() < 1 {
		return
	}

	// We must pop from the queue if possible
	// TODO: if this is a cold start, we either block the
	// scheduler or need a strategy
}

func (p *DefaultLocalPolicy) OnArrival(r *scheduledRequest) {
	containerID, err := node.AcquireWarmContainer(r.Fun)
	if err == nil {
		log.Printf("Using a warm container for: %v", r)
		execLocally(r, containerID, true)
	} else if errors.Is(err, node.NoWarmFoundErr) {
		if !handleColdStart(r) {
			dropRequest(r)
		}
	} else if errors.Is(err, node.OutOfResourcesErr) {
		log.Printf("Not enough resources.")

		// enqueue if possible
		if p.queue != nil && p.queue.Enqueue(r) {
			log.Printf("Added %v to queue (length=%d)", r, p.queue.Len())
		} else {
			dropRequest(r)
		}
	} else {
		// other error
		dropRequest(r)
	}
}
