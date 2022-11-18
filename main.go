package main

import (
	"fmt"
	"log"

	"github.com/snowmerak/hydro/broadcaster"
	"github.com/snowmerak/hydro/queue"
	"github.com/snowmerak/hydro/queue/ringbuffer"
)

func main() {
	bc := broadcaster.New(func(name string) queue.Queue[string] {
		return ringbuffer.New[string](name, 256)
	}, 8)
	bc.StartBroadcast()

	a, _ := bc.AddReceiver("A")
	b, _ := bc.AddReceiver("B")
	c, _ := bc.AddReceiver("C")
	_, ok := bc.AddReceiver("A") // must be nil and false
	if !ok {
		log.Println("A is already exists")
	}

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
