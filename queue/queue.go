package queue

type Queue[T any] interface {
	Send(T) error
	Receive() (T, error)
	Close()
	Name() string
}

type Sender[T any] interface {
	Send(T) error
}

type Receiver[T any] interface {
	Receive() (T, error)
	Name() string
}
