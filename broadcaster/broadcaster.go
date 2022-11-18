package broadcaster

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/snowmerak/hydro/gopool"
	"github.com/snowmerak/hydro/queue"
)

type Broadcaster[T any] struct {
	queueConstructor func(string) queue.Queue[T]

	queue     queue.Queue[T]
	receivers []queue.Queue[T]

	cancel chan struct{}

	interLock *sync.RWMutex

	pool *gopool.GoPool
}

func New[T any](queueConstructor func(name string) queue.Queue[T], workerSize int) *Broadcaster[T] {
	return &Broadcaster[T]{
		queueConstructor: queueConstructor,
		queue:            queueConstructor("main"),
		interLock:        &sync.RWMutex{},
		pool:             gopool.New(workerSize),
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
			select {
			case <-b.cancel:
				return
			default:
				v, err := b.queue.Receive()
				if err != nil {
					return
				}

				b.interLock.RLock()
				for _, r := range b.receivers {
					func(r queue.Queue[T]) {
						b.pool.Go(func() {
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
								log.Printf("broadcaster.StartBroadcast: %s was timeout\n", r.Name())
								b.RemoveReceiver(r)
							}
						})
					}(r)
				}
				b.interLock.RUnlock()
			}
		}
	}()
}

func (b *Broadcaster[T]) AddReceiver(name string) (queue.Receiver[T], bool) {
	b.interLock.Lock()
	defer b.interLock.Unlock()

	receiver := b.queueConstructor(name)

	if b.receivers == nil {
		b.receivers = make([]queue.Queue[T], 0)
	}

	if len(b.receivers) == 0 {
		b.receivers = append(b.receivers, receiver)
		return receiver, true
	}

	n, found := sort.Find(len(b.receivers), func(i int) int {
		target := b.receivers[i].Name()
		if target < name {
			return -1
		}
		if target > name {
			return 1
		}
		return 0
	})

	if found {
		receiver.Close()
		return nil, false
	}

	newReceivers := make([]queue.Queue[T], len(b.receivers)+1)
	copy(newReceivers, b.receivers[:n])
	newReceivers[n] = receiver
	copy(newReceivers[n+1:], b.receivers[n:])
	b.receivers = newReceivers

	return receiver, true
}

func (b *Broadcaster[T]) RemoveReceiver(receiver queue.Receiver[T]) {
	b.interLock.Lock()
	defer b.interLock.Unlock()

	i, found := sort.Find(len(b.receivers), func(i int) int {
		s := b.receivers[i].Name()
		t := receiver.Name()
		if s < t {
			return -1
		}
		if s > t {
			return 1
		}
		return 0
	})

	if !found {
		return
	}

	if i < len(b.receivers) && b.receivers[i].Name() == receiver.Name() {
		target := b.receivers[i]
		newReceivers := make([]queue.Queue[T], len(b.receivers)-1)
		copy(newReceivers, b.receivers[:i])
		copy(newReceivers[i:], b.receivers[i+1:])
		b.receivers = newReceivers
		target.Close()
	}
}

func (b *Broadcaster[T]) RemoveReceiverByName(name string) {
	b.interLock.Lock()
	defer b.interLock.Unlock()

	i, found := sort.Find(len(b.receivers), func(i int) int {
		s := b.receivers[i].Name()
		t := name
		if s < t {
			return -1
		}
		if s > t {
			return 1
		}
		return 0
	})

	if !found {
		return
	}

	if i < len(b.receivers) && b.receivers[i].Name() == name {
		receiver := b.receivers[i]
		newReceivers := make([]queue.Queue[T], len(b.receivers)-1)
		copy(newReceivers, b.receivers[:i])
		copy(newReceivers[i:], b.receivers[i+1:])
		b.receivers = newReceivers
		receiver.Close()
	}
}

func (b *Broadcaster[T]) Close() {
	b.interLock.Lock()
	defer b.interLock.Unlock()

	b.queue.Close()
	for _, r := range b.receivers {
		r.Close()
	}
	b.cancel <- struct{}{}
}
