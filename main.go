package main

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/snowmerak/hydro/broadcaster"
	"github.com/snowmerak/hydro/queue"
	"github.com/snowmerak/hydro/queue/ringbuffer"
)

func main() {
	const MAX = 100000

	bc := broadcaster.New(func(name string) queue.Queue[int64] {
		return ringbuffer.New[int64](name, 4096)
	}, 1024)
	bc.StartBroadcast()

	wg := new(sync.WaitGroup)
	for i := 0; i < 128; i++ {
		r, _ := bc.AddReceiver(strconv.FormatInt(int64(i), 10))
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				v, err := r.Receive()
				if err != nil {
					log.Println(err)
					return
				}
				if v == MAX {
					return
				}
			}
		}()
	}

	s := time.Now()

	for i := 1; i <= MAX; i++ {
		if err := bc.Send(int64(i)); err != nil {
			log.Println(err)
		}
	}

	wg.Wait()

	e := time.Now()

	fmt.Println(e.Sub(s))
}
