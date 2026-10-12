package bus

import (
	"context"
	"sync"
)

type route struct {
	subject string
	queue   string
}

type Event[T any] struct {
	adapter adapter[T]
	route   route
	bufSize int
	once    sync.Once
}

func NewEvent[T any](bufSize int, options ...Option) *Event[T] {
	e := &Event[T]{bufSize: bufSize}
	for _, option := range options {
		option(&e.route)
	}
	return e
}

func (e *Event[T]) Publish(ctx context.Context, message T) error {
	return e.resolve().Publish(ctx, message)
}

func (e *Event[T]) TryPublish(message T) error {
	return e.resolve().TryPublish(message)
}

func (e *Event[T]) Subscribe(bufSize int) (<-chan T, error) {
	return e.resolve().Subscribe(bufSize)
}

func (e *Event[T]) Unsubscribe(ch <-chan T) {
	e.resolve().Unsubscribe(ch)
}

func (e *Event[T]) Close(ctx context.Context) error {
	return e.resolve().Close(ctx)
}

func (e *Event[T]) resolve() adapter[T] {
	e.once.Do(func() {
		if e.route.subject == "" {
			e.adapter = newMemory[T](e.bufSize)
			return
		}
		driver := claimDriver()
		if driver == nil {
			e.adapter = newMemory[T](e.bufSize)
			return
		}
		e.adapter = newRemote[T](driver, e.route, e.bufSize)
	})
	return e.adapter
}
