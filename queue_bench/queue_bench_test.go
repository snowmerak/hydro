package queuebench_test

import (
	"runtime"
	"sync"
	"testing"

	"github.com/Workiva/go-datastructures/queue"
	genericqueue "github.com/lemon-mint/go-datastructures/queue"
)

const MAX = 100000000

func BenchmarkGoDatastructuresRingBuffer(b *testing.B) {
	q := queue.NewRingBuffer(128)
	writerGroup := new(sync.WaitGroup)
	for i := 0; i < 6; i++ {
		writerGroup.Add(1)
		go func() {
			defer writerGroup.Done()
			for i := 0; i < MAX; i++ {
				q.Put(i)
			}
		}()
	}
	readerGroup := new(sync.WaitGroup)
	for i := 0; i < 9; i++ {
		readerGroup.Add(1)
		go func() {
			defer readerGroup.Done()
			for !q.IsDisposed() {
				q.Get()
			}
		}()
	}
	writerGroup.Wait()
	q.Dispose()
	readerGroup.Wait()
	runtime.GC()
}

func BenchmarkGoDatastructuresGenericRingBuffer(b *testing.B) {
	q := genericqueue.NewRingBuffer[int](128)
	writerGroup := new(sync.WaitGroup)
	for i := 0; i < 6; i++ {
		writerGroup.Add(1)
		go func() {
			defer writerGroup.Done()
			for i := 0; i < MAX; i++ {
				q.Put(i)
			}
		}()
	}
	readerGroup := new(sync.WaitGroup)
	for i := 0; i < 9; i++ {
		readerGroup.Add(1)
		go func() {
			defer readerGroup.Done()
			for !q.IsDisposed() {
				q.Get()
			}
		}()
	}
	writerGroup.Wait()
	q.Dispose()
	readerGroup.Wait()
	runtime.GC()
}

func BenchmarkChannel(b *testing.B) {
	q := make(chan int, 128)
	writerGroup := new(sync.WaitGroup)
	for i := 0; i < 6; i++ {
		writerGroup.Add(1)
		go func() {
			defer writerGroup.Done()
			for i := 0; i < MAX; i++ {
				q <- i
			}
		}()
	}
	readerGroup := new(sync.WaitGroup)
	for i := 0; i < 9; i++ {
		readerGroup.Add(1)
		go func() {
			defer readerGroup.Done()
			for i := range q {
				_ = i
			}
		}()
	}
	writerGroup.Wait()
	close(q)
	readerGroup.Wait()
	runtime.GC()
}
