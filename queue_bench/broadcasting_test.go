package queuebench_test

import (
	"strconv"
	"sync"
	"testing"

	"github.com/snowmerak/hydro/broadcaster"
	"github.com/snowmerak/hydro/queue"
	"github.com/snowmerak/hydro/queue/ringbuffer"
)

func BenchmarkBroadcastring(b *testing.B) {
	const MAX = 1000

	bc := broadcaster.New(func(name string) queue.Queue[int64] {
		return ringbuffer.New[int64](name, 1024)
	}, 4096)
	bc.StartBroadcast()

	wg := new(sync.WaitGroup)
	for i := 0; i < 1024; i++ {
		r, _ := bc.AddReceiver(strconv.FormatInt(int64(i), 10))
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				v, err := r.Receive()
				if err != nil {
					b.Error(err)
					return
				}
				if v == MAX {
					return
				}
			}
		}()
	}

	for i := 1; i <= MAX; i++ {
		if err := bc.Send(int64(i)); err != nil {
			b.Fatal(err)
		}
	}

	wg.Wait()
}
