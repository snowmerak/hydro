package aero

import (
	"errors"
	"runtime"
	"sync/atomic"
)

var ErrOverrun = errors.New("subscriber overrun: data overwritten")

// PaddedCursor prevents false sharing on modern CPU cache lines.
type PaddedCursor struct {
	Val atomic.Int64
	_   [56]byte
}

// Aero is a high-performance Lock-Free Ring Buffer optimized for Fan-out event streaming.
// It uses bitwise masking for ultra-fast ring traversal and guarantees zero allocations along the fast path.
type Aero[T any] struct {
	buffer      []T
	capacity    int64
	mask        int64
	writeCursor PaddedCursor
}

// New creates a new Aero ring buffer rounded up to the nearest power of two capacity.
func New[T any](minCapacity int) *Aero[T] {
	var cap int64 = 1
	for cap < int64(minCapacity) {
		cap <<= 1
	}
	return &Aero[T]{
		buffer:   make([]T, cap),
		capacity: cap,
		mask:     cap - 1,
	}
}

// Publish stores the value into the ring buffer and increments the write cursor.
// For ultimate performance under massive bursts, it utilizes a Single Producer strategy.
func (a *Aero[T]) Publish(val T) {
	// 1. Read current sequence
	seq := a.writeCursor.Val.Load()
	
	// 2. Write data FIRST
	a.buffer[seq&a.mask] = val

	// 3. Increment cursor AFTER write
	a.writeCursor.Val.Store(seq + 1)
}

// Subscriber maintains an independent cursor to track its reading position within the Aero ring buffer.
type Subscriber[T any] struct {
	aero   *Aero[T]
	cursor int64
}

// Subscribe opens a new subscription tracking cursor starting from the current write head.
func (a *Aero[T]) Subscribe() *Subscriber[T] {
	return &Subscriber[T]{
		aero:   a,
		cursor: a.writeCursor.Val.Load(),
	}
}

// Receive blocks via adaptive spinning until a new event is available.
// In the event of an overrun (producer overrides the unread cursor), it fast-forwards 
// to the oldest valid data and returns ErrOverrun as a warning.
func (s *Subscriber[T]) Receive() (T, error) {
	target := s.cursor
	var currentWrite int64

	// Adaptive Spin to wait for the publisher
	for {
		currentWrite = s.aero.writeCursor.Val.Load()
		if currentWrite > target {
			break
		}
		runtime.Gosched()
	}

	var err error
	// Overrun detection (Lossy logic: Option 2)
	if currentWrite-target > s.aero.capacity {
		// Target has been overwritten. Recover by skipping to the oldest alive sequence.
		// (The write head minus capacity is the oldest element that hasn't been overwritten yet).
		target = currentWrite - s.aero.capacity
		err = ErrOverrun
	}

	val := s.aero.buffer[target&s.aero.mask]
	s.cursor = target + 1
	return val, err
}
