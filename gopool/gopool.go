package gopool

import (
	"log"
	"runtime"
	"sync"
	"sync/atomic"
)

type GoPool struct {
	list      []chan parameter
	isRunning []bool
	running   int64
	sync.Mutex
}

type parameter func()

func do(gp *GoPool, ch chan parameter, index int) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("gopool: %v", err)
		}
		gp.Lock()
		gp.running--
		gp.isRunning[index] = false
		gp.Unlock()
		go do(gp, ch, index)
	}()
	for p := range ch {
		p()
		gp.Lock()
		gp.running--
		gp.isRunning[index] = false
		gp.Unlock()
	}
}

func New(max int) *GoPool {
	gp := new(GoPool)
	gp.list = make([]chan parameter, max)
	for i := 0; i < max; i++ {
		gp.list[i] = make(chan parameter)
		go do(gp, gp.list[i], i)
	}
	gp.isRunning = make([]bool, max)
	gp.running = 0
	return gp
}

func (gp *GoPool) Go(f func()) error {
	for {
		gp.Lock()
		for i := 0; i < len(gp.list); i++ {
			if !gp.isRunning[i] {
				gp.isRunning[i] = true
				gp.list[i] <- f
				gp.running++
				gp.Unlock()
				return nil
			}
		}
		gp.Unlock()
		runtime.Gosched()
	}
}

func (gp *GoPool) Wait() {
	for atomic.LoadInt64(&gp.running) > 0 {
		runtime.Gosched()
	}
}

func (gp *GoPool) Close() {
	gp.Lock()
	defer gp.Unlock()
	for i := 0; i < len(gp.list); i++ {
		close(gp.list[i])
	}
}
