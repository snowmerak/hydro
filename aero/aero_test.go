package aero

import (
	"sync"
	"testing"
)

func TestAero_SinglePublishAndReceive(t *testing.T) {
	a := New[int](16)
	sub := a.Subscribe()

	a.Publish(42)
	val, err := sub.Receive()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if val != 42 {
		t.Fatalf("Expected 42, but got %d", val)
	}
}

func TestAero_Overrun(t *testing.T) {
	a := New[int](4) // capacity will be 4
	sub := a.Subscribe()

	// Publish 5 items. The first item (1) will be overwritten by the 5th item (5).
	a.Publish(1)
	a.Publish(2)
	a.Publish(3)
	a.Publish(4)
	a.Publish(5)

	// Since we fell behind by more than 4, Receive should detect overrun 
	// and fast-forward to the oldest valid item (which is 2).
	val, err := sub.Receive()
	if err != ErrOverrun {
		t.Fatalf("Expected ErrOverrun, got %v", err)
	}
	if val != 2 {
		t.Fatalf("Expected 2 after overrun recovery, got %d", val)
	}
}

func TestAero_MultiSubscribers(t *testing.T) {
	a := New[int](1024)
	
	numSubs := 50
	subs := make([]*Subscriber[int], numSubs)
	for i := 0; i < numSubs; i++ {
		subs[i] = a.Subscribe()
	}

	a.Publish(999)

	var wg sync.WaitGroup
	wg.Add(numSubs)
	for i := 0; i < numSubs; i++ {
		go func(sub *Subscriber[int]) {
			defer wg.Done()
			val, _ := sub.Receive()
			if val != 999 {
				t.Errorf("MultiSub Failed: %v", val)
			}
		}(subs[i])
	}
	wg.Wait()
}

func TestAero_HighConcurrencySingleProducer(t *testing.T) {
	a := New[int](65536) // Large enough to prevent overrun for testing
	numSubs := 5
	msgCount := 10000

	var subs []*Subscriber[int]
	for i := 0; i < numSubs; i++ {
		subs = append(subs, a.Subscribe())
	}

	var wg sync.WaitGroup
	wg.Add(numSubs)

	for i := 0; i < numSubs; i++ {
		go func(sub *Subscriber[int]) {
			defer wg.Done()
			for expected := 1; expected <= msgCount; expected++ {
				val, err := sub.Receive()
				if err != nil {
					t.Errorf("Unexpected overrun: %v", err)
					return
				}
				if val != expected {
					t.Errorf("Streaming Mismatch: expected %d, got %d", expected, val)
					return
				}
			}
		}(subs[i])
	}

	go func() {
		for i := 1; i <= msgCount; i++ {
			a.Publish(i)
		}
	}()

	wg.Wait()
}

func BenchmarkAero_SinglePublish(b *testing.B) {
	a := New[int](1024 * 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Publish(i)
	}
}

func BenchmarkAero_PublishAndReceive(b *testing.B) {
	a := New[int](1024 * 1024 * 64) // Prevent overrun
	sub := a.Subscribe()
	b.ResetTimer()

	go func() {
		for i := 0; i < b.N; i++ {
			a.Publish(i)
		}
	}()

	for i := 0; i < b.N; i++ {
		sub.Receive()
	}
}

func BenchmarkAero_MultiSubscriber(b *testing.B) {
	a := New[int](1024 * 1024 * 64)
	numSubs := 5
	var subs []*Subscriber[int]
	for i := 0; i < numSubs; i++ {
		subs = append(subs, a.Subscribe())
	}
	
	b.ResetTimer()

	go func() {
		for i := 0; i < b.N; i++ {
			a.Publish(i)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(numSubs)
	for i := 0; i < numSubs; i++ {
		go func(s *Subscriber[int]) {
			defer wg.Done()
			for j := 0; j < b.N; j++ {
				s.Receive()
			}
		}(subs[i])
	}
	wg.Wait()
}
