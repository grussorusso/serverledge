package scheduling

type queue interface {
	Enqueue(r *scheduledRequest) bool
	Dequeue() *scheduledRequest
	Len() int
}

// FIFOQueue defines a circular queue
type FIFOQueue struct {
	data     []*scheduledRequest
	capacity int
	head     int
	tail     int
	size     int
}

// NewFIFOQueue creates a queue
func NewFIFOQueue(n int) *FIFOQueue {
	if n == 0 {
		return nil
	}
	return &FIFOQueue{
		data:     make([]*scheduledRequest, n),
		capacity: n,
		head:     0,
		tail:     0,
		size:     0,
	}
}

// IsEmpty returns true if queue is empty
func (q *FIFOQueue) IsEmpty() bool {
	if q.head == q.tail {
		return true
	}
	return false
}

// IsFull returns true if queue is full
func (q *FIFOQueue) IsFull() bool {
	if q.head == (q.tail+1)%q.capacity {
		return true
	}
	return false
}

// Enqueue pushes an element to the back
func (q *FIFOQueue) Enqueue(v *scheduledRequest) bool {
	if q.IsFull() {
		return false
	}

	q.data[q.tail] = v
	q.tail = (q.tail + 1) % q.capacity
	q.size = q.size + 1
	return true
}

// Dequeue fetches a element from queue
func (q *FIFOQueue) Dequeue() interface{} {
	if q.IsEmpty() {
		return nil
	}
	v := q.data[q.head]
	q.head = (q.head + 1) % q.capacity
	q.size = q.size - 1
	return v
}

// Returns the current length of the queue
func (q *FIFOQueue) Len() int {
	return q.size
}
