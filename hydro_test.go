package hydro

import (
	"sync"
	"testing"
	"time"
)

// 1. Basic verification for single item publish/receive
func TestHydro_SinglePublishAndReceive(t *testing.T) {
	h := New[int]()
	sub := h.Subscribe()

	h.Publish(42)
	val := sub.Receive()
	if val != 42 {
		t.Fatalf("Single Expected 42, but got %d", val)
	}
}

// 2. Verification for variadic batch publish
func TestHydro_BatchPublishAndReceive(t *testing.T) {
	h := New[string]()
	sub := h.Subscribe()

	h.Publish("A", "B", "C")

	if val := sub.Receive(); val != "A" {
		t.Fatalf("Batch Failed: %v", val)
	}
	if val := sub.Receive(); val != "B" {
		t.Fatalf("Batch Failed: %v", val)
	}
	if val := sub.Receive(); val != "C" {
		t.Fatalf("Batch Failed: %v", val)
	}
}

// 3. Verification for streams mixing single and batched publishes
func TestHydro_MixedPublish(t *testing.T) {
	h := New[int]()
	sub := h.Subscribe()

	h.Publish(1)
	h.Publish(2, 3, 4)
	h.Publish(5)

	expected := []int{1, 2, 3, 4, 5}
	for _, exp := range expected {
		if val := sub.Receive(); val != exp {
			t.Fatalf("Mixed Expected %d, but got %d", exp, val)
		}
	}
}

// 4. Verification for no-op exception handling when Publish is called with zero arguments
func TestHydro_EmptyPublish(t *testing.T) {
	h := New[int]()
	sub := h.Subscribe()

	h.Publish() // Should be a no-op

	// This empty publish does not add a node, so the receiver should remain blocked.
	done := make(chan struct{})
	go func() {
		sub.Receive()
		close(done)
	}()

	select {
	case <-done:
		t.Fatalf("Empty Publish caused false wakeup and unblocked Receive")
	case <-time.After(50 * time.Millisecond):
		// If it remains blocked for 50ms, it is considered a Pass.
	}
}

// 5. Verifies logic securing only the latest tail data for late subscribers
func TestHydro_LateSubscriber(t *testing.T) {
	h := New[int]()
	sub1 := h.Subscribe()

	h.Publish(1, 2, 3)

	// The second subscriber joins after the first 3 items have been dispatched
	sub2 := h.Subscribe()
	h.Publish(4, 5)

	// The first subscriber must receive all items
	for i := 1; i <= 5; i++ {
		if val := sub1.Receive(); val != i {
			t.Fatalf("Sub1 Result Mismatch: expected %d, got %d", i, val)
		}
	}

	// The late subscriber must only receive data published after its subscription (4, 5)
	for i := 4; i <= 5; i++ {
		if val := sub2.Receive(); val != i {
			t.Fatalf("Sub2 Result Mismatch: expected %d, got %d", i, val)
		}
	}
}

// 6. Verification for massive multi-subscriber concurrency (Fan-out Broadcasting)
func TestHydro_MultiSubscribers(t *testing.T) {
	h := New[int]()

	numSubs := 100
	subs := make([]*Subscriber[int], numSubs)
	for i := 0; i < numSubs; i++ {
		subs[i] = h.Subscribe()
	}

	// Waits for all 100 goroutines to acquire the target value upon a single publish event
	h.Publish(999)

	var wg sync.WaitGroup
	wg.Add(numSubs)
	for i := 0; i < numSubs; i++ {
		go func(sub *Subscriber[int]) {
			defer wg.Done()
			if val := sub.Receive(); val != 999 {
				t.Errorf("MultiSub Failed: %v", val)
			}
		}(subs[i])
	}
	wg.Wait()
}

// 7. Single producer extreme workload test to verify prevention of lost wakeups/missed updates
func TestHydro_HighConcurrencySingleProducer(t *testing.T) {
	h := New[int]()
	numSubs := 5
	msgCount := 10000

	var subs []*Subscriber[int]
	for i := 0; i < numSubs; i++ {
		subs = append(subs, h.Subscribe())
	}

	var wg sync.WaitGroup
	wg.Add(numSubs)

	for i := 0; i < numSubs; i++ {
		go func(sub *Subscriber[int]) {
			defer wg.Done()
			for expected := 1; expected <= msgCount; expected++ {
				val := sub.Receive()
				if val != expected {
					t.Errorf("Streaming Mismatch: expected %d, got %d", expected, val)
					return // Disconnect to prevent log spam
				}
			}
		}(subs[i])
	}

	go func() {
		// Publisher logic: Dynamically publish a mix of single items and batches of 5 at random intervals
		for i := 1; i <= msgCount; {
			if i%7 == 0 && i+4 <= msgCount {
				h.Publish(i, i+1, i+2, i+3, i+4)
				i += 5
			} else {
				h.Publish(i)
				i++
			}
		}
	}()

	wg.Wait()
}

// 8. Prove Compare-And-Swap (CAS) integrity and thread safety under competitive multi-producer conditions
func TestHydro_HighConcurrencyMultiProducer(t *testing.T) {
	h := New[int]()
	numProducers := 10
	numMsgs := 1000
	totalMsgs := numProducers * numMsgs

	sub := h.Subscribe()

	var wg sync.WaitGroup
	wg.Add(numProducers)

	for p := 0; p < numProducers; p++ {
		go func(pid int) {
			defer wg.Done()
			for i := 0; i < numMsgs; i++ {
				h.Publish(pid*10000 + i) // Insert unique value
			}
		}(p)
	}

	go func() {
		wg.Wait()
	}()

	// The total processed count must perfectly match totalMsgs despite CAS contention among multi-writers
	received := make(map[int]bool)
	for i := 0; i < totalMsgs; i++ {
		val := sub.Receive()
		received[val] = true
	}

	if len(received) != totalMsgs {
		t.Fatalf("Total Count Mismatch: expected %d, actual %d", totalMsgs, len(received))
	}
}
