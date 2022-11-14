package queue

type Queue[T any] interface {
	Send(T) error
	Receive() (T, error)
	Close()
	Name() string
}
