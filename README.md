# Hydro

Hydro is a high-performance, lock-free fan-out data structure implemented in Go. It supports mixed traffic of single and batched operations for optimal latency and throughput under high-contention scenarios.

## Features

*   **Lock-Free Architecture**: Utilizes Compare-And-Swap (CAS) operations to avoid thread blocking during multi-producer scenarios.
*   **Adaptive Spin**: Consumers employ a short adaptive spin loop before falling back to channel-based waits, reducing active context switches during high-speed sequential processing.
*   **Batch Node Linking**: Combines variadic batch publishes into chunked, singly linked list nodes for cache locality and minimal memory allocation overhead.
*   **Zero-Overhead Local Fetch**: Subscribers maintain active cursors within node chunks, accessing batch elements via index increments.
*   **Late Subscription Support**: New subscribers independently track positions, efficiently skipping historical records and starting from live data at the moment of subscription.

## Usage

```go
package main

import (
	"fmt"
	"github.com/snowmerak/hydro"
)

func main() {
	// 1. Initialize a new stream
	stream := hydro.New[string]()

	// 2. Add subscribers
	sub1 := stream.Subscribe()
	sub2 := stream.Subscribe()

	// 3. Publish data (Single or Batched)
	stream.Publish("event1")
	stream.Publish("event2", "event3")

	// 4. Receive data
	fmt.Println("Sub1 received:", sub1.Receive()) // "event1"
	fmt.Println("Sub2 received:", sub2.Receive()) // "event1"
}
```

## Testing

Extensive unit tests are provided in `hydro_test.go` covering edge cases.
Execute the testing suite via:

```sh
go test -v ./...
```

### Coverage
The implementation holds **100.0%** statement coverage.

```sh
$ go test -cover .
ok      github.com/snowmerak/hydro      0.612s  coverage: 100.0% of statements
```

## Performance

Hydro is deeply optimized. The benchmarks (run on an AMD Ryzen 7 8845HS) demonstrate minimal allocations and microsecond-latency profiles across various loads:

```
goos: windows
goarch: amd64
pkg: github.com/snowmerak/hydro
cpu: AMD Ryzen 7 8845HS w/ Radeon 780M Graphics
BenchmarkHydro_SinglePublish-16         14356075                86.54 ns/op          168 B/op          3 allocs/op
BenchmarkHydro_BatchPublish-16          11945670               109.0 ns/op           240 B/op          3 allocs/op
BenchmarkHydro_PublishAndReceive-16      6976261               181.7 ns/op           168 B/op          3 allocs/op
BenchmarkHydro_MultiSubscriber-16        5323629               223.9 ns/op           168 B/op          3 allocs/op
```
