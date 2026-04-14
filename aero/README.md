# Aero

Aero is a hyper-optimized, zero-allocation ring buffer (LMAX Disruptor pattern) designed for screaming-fast fan-out event streaming in Go. By allowing a Single Producer Multiple Consumer (SPMC) structure with lossy overrun protection, it achieves nano-second latency throughput.

## Features

*   **Zero Allocations**: Fully pre-allocated ring buffer guaranteeing `0 allocs/op` on the direct execution path.
*   **False Sharing Prevention**: `PaddedCursor` structs are padded with 64-bytes to ensure CPU cache lines do not overlap or induce hardware contention.
*   **Bitwise Optimization**: Traversal and array boundaries are managed via ultra-fast bitwise masks (`capacity - 1`) utilizing powers of two.
*   **SPMC Centric**: Highly optimized Single Producer Multiple Consumer memory access pattern with asynchronous read-cursors.
*   **Lossy Overrun Protection**: When a consumer thread falls behind the publisher's boundary, it automatically fast-forwards to the oldest surviving data and returns an `ErrOverrun` signal instead of blocking the publisher.

## Usage

```go
package main

import (
	"fmt"
	"github.com/snowmerak/hydro/aero"
)

func main() {
	// 1. Initialize a new stream (Size must be a power of 2, e.g., 1024)
	stream := aero.New[string](1024)

	// 2. Add subscribers
	sub1 := stream.Subscribe()
	sub2 := stream.Subscribe()

	// 3. Publish data (Single Publisher recommended for peak performance)
	stream.Publish("event1")
	stream.Publish("event2")

	// 4. Receive data
	// If the subscriber is too slow, it returns aero.ErrOverrun and skips lost items.
	if val, err := sub1.Receive(); err == nil {
		fmt.Println("Sub1 received:", val) // "event1"
	}
	
	val, _ := sub2.Receive()
	fmt.Println("Sub2 received:", val) // "event1"
}
```

## Testing

Execute the test suite via the Go command line:

```sh
go test -v ./...
```

### Coverage
The core logic maintains **100.0%** unit test statement coverage.

```sh
$ go test -cover .
ok      github.com/snowmerak/hydro/aero      0.460s  coverage: 100.0% of statements
```

## Performance

The codebase is engineered strictly for speed. Below are the benchmark outputs recorded on an AMD Ryzen 7 8845HS:

```
goos: windows
goarch: amd64
pkg: github.com/snowmerak/hydro/aero
cpu: AMD Ryzen 7 8845HS w/ Radeon 780M Graphics
BenchmarkAero_SinglePublish-16          597415876         1.997 ns/op        0 B/op        0 allocs/op
BenchmarkAero_PublishAndReceive-16      127940403        10.88 ns/op         0 B/op        0 allocs/op
BenchmarkAero_MultiSubscriber-16         18193726        66.63 ns/op         0 B/op        0 allocs/op
```
