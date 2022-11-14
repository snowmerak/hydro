package ringbuffer

import (
	"fmt"

	"github.com/lemon-mint/go-datastructures/queue"
)

type RingBuffer[T any] struct {
	buffer *queue.RingBuffer[T]
	name   string
}

func New[T any](name string, size uint64) *RingBuffer[T] {
	return &RingBuffer[T]{
		buffer: queue.NewRingBuffer[T](size),
		name:   name,
	}
}

func (r *RingBuffer[T]) Send(value T) error {
	if err := r.buffer.Put(value); err != nil {
		return fmt.Errorf("ringbuffer.Send: %w", err)
	}
	return nil
}

func (r *RingBuffer[T]) Receive() (T, error) {
	v, err := r.buffer.Get()
	if err != nil {
		return v, fmt.Errorf("ringbuffer.Receive: %w", err)
	}
	return v, nil
}

func (r *RingBuffer[T]) Close() {
	r.buffer.Dispose()
}

func (r *RingBuffer[T]) Name() string {
	return r.name
}
