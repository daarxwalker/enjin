package bus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

type remote[T any] struct {
	driver       Driver
	subscription Subscription
	local        *memory[T]
	route        route
	mu           sync.RWMutex
	closed       bool
}

func newRemote[T any](driver Driver, eventRoute route, bufSize int) *remote[T] {
	return &remote[T]{
		driver: driver,
		route:  eventRoute,
		local:  newMemory[T](bufSize),
	}
}

func (r *remote[T]) Publish(ctx context.Context, message T) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return ErrClosed
	}
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("bus: encode %s failed: %w", r.route.subject, err)
	}
	return r.driver.Publish(ctx, r.route.subject, data)
}

func (r *remote[T]) TryPublish(message T) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.closed {
		return ErrClosed
	}
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("bus: encode %s failed: %w", r.route.subject, err)
	}
	return r.driver.TryPublish(r.route.subject, data)
}

func (r *remote[T]) Subscribe(bufSize int) (<-chan T, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.closed && r.subscription == nil {
		subscription, err := r.driver.Subscribe(r.route.subject, r.route.queue, r.receive)
		if err != nil {
			ch := make(chan T)
			close(ch)
			return ch, fmt.Errorf("bus: subscribe %s failed: %w", r.route.subject, err)
		}
		r.subscription = subscription
	}
	return r.local.Subscribe(bufSize)
}

func (r *remote[T]) Unsubscribe(ch <-chan T) {
	r.local.Unsubscribe(ch)
}

func (r *remote[T]) Close(ctx context.Context) error {
	r.mu.Lock()
	r.closed = true
	subscription := r.subscription
	r.subscription = nil
	r.mu.Unlock()
	var drainErr error
	if subscription != nil {
		drainErr = subscription.Close(ctx)
	}
	return errors.Join(drainErr, r.local.Close(ctx))
}

func (r *remote[T]) receive(ctx context.Context, data []byte) error {
	var message T
	if err := json.Unmarshal(data, &message); err != nil {
		return fmt.Errorf("bus: decode %s failed: %w", r.route.subject, err)
	}
	return r.local.Publish(ctx, message)
}
