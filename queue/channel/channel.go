package channel

import (
	"errors"
	"sync/atomic"
	"time"
)

type Channel[T any] struct {
	channel chan T
	closed  atomic.Bool
	name    string
}

func New[T any](name string, size int) *Channel[T] {
	return &Channel[T]{
		channel: make(chan T, size),
		closed:  atomic.Bool{},
		name:    name,
	}
}

func (c *Channel[T]) Send(value T) error {
	if c.closed.Load() {
		return errors.New("Channel.Send: channel is closed")
	}
	c.channel <- value
	return nil
}

func (c *Channel[T]) Receive() (T, error) {
	v, ok := <-c.channel
	if !ok {
		return v, errors.New("Channel.Receive: channel closed")
	}
	return v, nil
}

func (c *Channel[T]) Close() {
	c.closed.Store(true)
	time.Sleep(time.Millisecond)
	close(c.channel)
}

func (c *Channel[T]) Name() string {
	return c.name
}
