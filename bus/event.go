package bus

import (
	"context"
	"fmt"
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
	if err := e.resolve().Publish(ctx, message); err != nil {
		return fmt.Errorf("publish event failed: %w", err)
	}
	return nil
}

func (e *Event[T]) TryPublish(message T) error {
	if err := e.resolve().TryPublish(message); err != nil {
		return fmt.Errorf("try publish event failed: %w", err)
	}
	return nil
}

func (e *Event[T]) Subscribe(bufSize int) (<-chan T, error) {
	ch, err := e.resolve().Subscribe(bufSize)
	if err != nil {
		return ch, fmt.Errorf("subscribe to event failed: %w", err)
	}
	return ch, nil
}

func (e *Event[T]) Unsubscribe(ch <-chan T) {
	e.resolve().Unsubscribe(ch)
}

func (e *Event[T]) Close(ctx context.Context) error {
	if err := e.resolve().Close(ctx); err != nil {
		return fmt.Errorf("close event failed: %w", err)
	}
	return nil
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
