package broadcaster

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/snowmerak/hydro/queue"
)

type Broadcaster[T any] struct {
	queueConstructor func(string) queue.Queue[T]

	queue     queue.Queue[T]
	receivers []queue.Queue[T]

	interLock *sync.RWMutex
}

func New[T any](queueConstructor func(name string) queue.Queue[T]) *Broadcaster[T] {
	return &Broadcaster[T]{
		queueConstructor: queueConstructor,
		queue:            queueConstructor("main"),
		interLock:        &sync.RWMutex{},
	}
}

func (b *Broadcaster[T]) Send(value T) error {
	if err := b.queue.Send(value); err != nil {
		return fmt.Errorf("broadcaster.Send: %w", err)
	}

	return nil
}

func (b *Broadcaster[T]) StartBroadcast() {
	go func() {
		for {
			v, err := b.queue.Receive()
			if err != nil {
				return
			}

			b.interLock.RLock()
			for _, r := range b.receivers {
				fmt.Println(r)
				go func(r queue.Queue[T]) {
					ch := make(chan error)
					go func() {
						defer func() {
							if r := recover(); r != nil {
								ch <- fmt.Errorf("broadcaster.StartBroadcast: %v", r)
							}
							close(ch)
						}()
						ch <- r.Send(v)
					}()
					after := time.After(5 * time.Second)
					select {
					case v := <-ch:
						if v != nil {
							log.Println("broadcaster.StartBroadcast: ", v)
						}
					case <-after:
						log.Println("broadcaster.StartBroadcast: timeout")
					}
				}(r)
			}
			b.interLock.RUnlock()
		}
	}()
}

func (b *Broadcaster[T]) AddReceiver(name string) queue.Queue[T] {
	b.interLock.Lock()
	defer b.interLock.Unlock()

	receiver := b.queueConstructor(name)

	if b.receivers == nil {
		b.receivers = make([]queue.Queue[T], 0)
	}

	if len(b.receivers) == 0 {
		b.receivers = append(b.receivers, receiver)
		return receiver
	}

	n := sort.Search(len(b.receivers), func(i int) bool {
		return b.receivers[i].Name() < name
	})
	newReceivers := make([]queue.Queue[T], len(b.receivers)+1)
	copy(newReceivers, b.receivers[:n])
	newReceivers[n] = receiver
	copy(newReceivers[n+1:], b.receivers[n:])
	b.receivers = newReceivers

	return receiver
}

func (b *Broadcaster[T]) RemoveReceiver(receiver queue.Queue[T]) {
	b.interLock.Lock()
	defer b.interLock.Unlock()

	i := sort.Search(len(b.receivers), func(i int) bool {
		return b.receivers[i].Name() < receiver.Name()
	})

	if i < len(b.receivers) && b.receivers[i].Name() == receiver.Name() {
		b.receivers = append(b.receivers[:i], b.receivers[i+1:]...)
	}
}

func (b *Broadcaster[T]) RemoveReceiverByName(name string) {
	b.interLock.Lock()
	defer b.interLock.Unlock()

	i := sort.Search(len(b.receivers), func(i int) bool {
		return b.receivers[i].Name() < name
	})

	if i < len(b.receivers) && b.receivers[i].Name() == name {
		b.receivers = append(b.receivers[:i], b.receivers[i+1:]...)
	}
}
