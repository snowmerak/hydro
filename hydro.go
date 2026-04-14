package hydro

import (
	"runtime"
	"sync/atomic"
)

// Hydro is a high-performance, lock-free fan-out data structure that utilizes
// Adaptive Spin and Batch Node linking techniques.
// It flexibly maintains streaming performance under mixed traffic of single and batched operations.
type Hydro[T any] struct {
	tail atomic.Pointer[Node[T]]
}

// Node is an element of the chunked singly linked list that accommodates variable-sized data.
type Node[T any] struct {
	Values []T
	Next   atomic.Pointer[Node[T]]
	wait   chan struct{}
}

// New creates and returns a new empty Hydro event stream.
func New[T any]() *Hydro[T] {
	head := &Node[T]{
		wait: make(chan struct{}),
	}
	h := &Hydro[T]{}
	h.tail.Store(head)
	return h
}

// Publish takes one or more values, wraps them in a new single chunk node,
// and appends it to the tail.
func (h *Hydro[T]) Publish(vals ...T) {
	if len(vals) == 0 {
		return
	}

	newNode := &Node[T]{
		Values: vals,
		wait:   make(chan struct{}),
	}

	for {
		oldTail := h.tail.Load()
		if h.tail.CompareAndSwap(oldTail, newNode) {
			oldTail.Next.Store(newNode)
			close(oldTail.wait)
			break
		}
	}
}

// Subscriber is an object that independently maintains its cursor position within the Hydro stream.
type Subscriber[T any] struct {
	current *Node[T]
	index   int
}

// Subscribe returns a subscriber starting from the latest tail at the time of invocation.
// It skips past historical data and begins streaming from future published data.
func (h *Hydro[T]) Subscribe() *Subscriber[T] {
	t := h.tail.Load()
	return &Subscriber[T]{
		current: t,
		index:   len(t.Values), // Skip already published elements in the current chunk
	}
}

// Receive blocks and returns the next event value using a pure lock-free approach.
// It features an internal Adaptive Spin to actively prevent context switches (system calls)
// during high-speed sequential processing.
func (s *Subscriber[T]) Receive() T {
	// 1. If there are remaining elements in the current chunk, retrieve them immediately
	// by incrementing the index (Zero-overhead LockFree).
	if s.index < len(s.current.Values) {
		val := s.current.Values[s.index]
		s.index++
		return val
	}

	// 2. Adaptive Spin Loop (Proactive peeking)
	// Spin for 50 cycles to detect if the next node has been attached via Publish(),
	// avoiding immediate channel queue blocking.
	var nextNode *Node[T]
	for i := 0; i < 50; i++ {
		nextNode = s.current.Next.Load()
		if nextNode != nil {
			break
		}
		runtime.Gosched()
	}

	// 3. Fallback: If not acquired during the short spin, block on the physical thread
	// using the wait channel (Optimizes idle CPU usage).
	if nextNode == nil {
		<-s.current.wait
		nextNode = s.current.Next.Load()
	}

	// 4. Move the cursor completely to the next node.
	s.current = nextNode
	s.index = 1 // Instead of starting from 0 on the next call, extract the 0th value now and set index to 1
	return nextNode.Values[0]
}
