# Hydro

hydro is a broadcaster for a process written go.

## install

`go get github.com/snowmerak/hydro`

## use

```go
package main

import (
	"fmt"

	"github.com/snowmerak/hydro/broadcaster"
	"github.com/snowmerak/hydro/queue"
	"github.com/snowmerak/hydro/queue/ringbuffer"
)

func main() {
	bc := broadcaster.New(func(name string) queue.Queue[string] {
		return ringbuffer.New[string](name, 256)
	})
	bc.StartBroadcast()

	a := bc.AddReceiver("A")
	b := bc.AddReceiver("B")
	c := bc.AddReceiver("C")

	bc.Send("Hello, World!")

	fmt.Println(a.Receive())
	fmt.Println(b.Receive())
	fmt.Println(c.Receive())

	bc.RemoveReceiver(b)
	bc.RemoveReceiverByName("C")

	bc.Send("Hello, World!")

	fmt.Println(a.Receive())
	fmt.Println(b.Receive())
	fmt.Println(c.Receive())
}
```

```zsh
➜  hydro git:(main) ✗ go run .                 
Hello, World! <nil>
Hello, World! <nil>
Hello, World! <nil>
Hello, World! <nil>
 ringbuffer.Receive: queue: disposed
 ringbuffer.Receive: queue: disposed
```