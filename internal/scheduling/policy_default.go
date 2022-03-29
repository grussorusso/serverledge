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
	if queueCapacity > 0 {
		log.Printf("Configured queue with capacity %d", queueCapacity)
		p.queue = NewFIFOQueue(queueCapacity)
	} else {
		p.queue = nil
	}
}

func (p *DefaultLocalPolicy) OnCompletion(completed *scheduledRequest) {
	if p.queue == nil {
		return
	}

	p.queue.Lock()
	defer p.queue.Unlock()
	if p.queue.Len() == 0 {
		return
	}

	req := p.queue.Front()

	containerID, err := node.AcquireWarmContainer(req.Fun)
	if err == nil {
		p.queue.Dequeue()
		log.Printf("[%s] Warm start from the queue (length=%d)", req, p.queue.Len())
		execLocally(req, containerID, true)
		return
	}

	if errors.Is(err, node.NoWarmFoundErr) {
		if handleColdStart(req) { // TODO: this will block the thread and the queue
			log.Printf("Cold start from the queue: %v", *req)
			p.queue.Dequeue()
			return
		}
	} else if errors.Is(err, node.OutOfResourcesErr) {
	} else {
		// other error
		p.queue.Dequeue()
		dropRequest(req)
	}
}

func (p *DefaultLocalPolicy) OnArrival(r *scheduledRequest) {
	log.Printf("[%s] OnArrival", *r)
	containerID, err := node.AcquireWarmContainer(r.Fun)
	if err == nil {
		log.Printf("[%s] Warm", *r)
		execLocally(r, containerID, true)
		return
	}

	if errors.Is(err, node.NoWarmFoundErr) {
		log.Printf("[%s] Trying Cold Start", *r)
		if handleColdStart(r) {
			log.Printf("[%s] Cold Start", *r)
			return
		}
	} else if errors.Is(err, node.OutOfResourcesErr) {
		// pass
	} else {
		// other error
		dropRequest(r)
		return
	}

	// enqueue if possible
	if p.queue != nil {
		log.Printf("[%s] Possible queueing", *r)
		p.queue.Lock()
		defer p.queue.Unlock()
		if p.queue.Enqueue(r) {
			log.Printf("Added %v to queue (length=%d)", r, p.queue.Len())
			return
		}
	}

	log.Printf("[%s] Nothing left but drop", *r)
	dropRequest(r)
}
