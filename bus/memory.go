package bus

import (
	"context"
	"sync"
)

type memory[T any] struct {
	publish     chan T
	closing     chan struct{}
	abort       chan struct{}
	stopped     chan struct{}
	subscribers []*subscriber[T]
	mu          sync.RWMutex
	closeOnce   sync.Once
	abortOnce   sync.Once
	subMu       sync.Mutex
	closed      bool
	subClosed   bool
}

func newMemory[T any](bufSize int) *memory[T] {
	m := &memory[T]{
		publish: make(chan T, bufSize),
		closing: make(chan struct{}),
		abort:   make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go m.dispatch()
	return m
}

func (m *memory[T]) Publish(ctx context.Context, message T) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return ErrClosed
	}
	select {
	case m.publish <- message:
		return nil
	case <-m.closing:
		return ErrClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *memory[T]) TryPublish(message T) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return ErrClosed
	}
	select {
	case m.publish <- message:
		return nil
	default:
		return ErrFull
	}
}

func (m *memory[T]) Subscribe(bufSize int) (<-chan T, error) {
	sub := newSubscriber[T](bufSize)
	m.subMu.Lock()
	defer m.subMu.Unlock()
	if m.subClosed {
		sub.close()
		return sub.ch, ErrClosed
	}
	m.subscribers = append(m.subscribers, sub)
	return sub.ch, nil
}

func (m *memory[T]) Unsubscribe(ch <-chan T) {
	m.subMu.Lock()
	var found *subscriber[T]
	for i, sub := range m.subscribers {
		if sub.ch == ch {
			found = sub
			m.subscribers = append(m.subscribers[:i], m.subscribers[i+1:]...)
			break
		}
	}
	m.subMu.Unlock()
	if found != nil {
		found.close()
	}
}

func (m *memory[T]) Close(ctx context.Context) error {
	m.closeOnce.Do(func() {
		close(m.closing)
		m.mu.Lock()
		m.closed = true
		close(m.publish)
		m.mu.Unlock()
	})
	select {
	case <-m.stopped:
		return nil
	case <-ctx.Done():
		m.abortOnce.Do(func() {
			close(m.abort)
		})
		<-m.stopped
		return ctx.Err()
	}
}

func (m *memory[T]) dispatch() {
	defer close(m.stopped)
	defer m.shutdown()
	for message := range m.publish {
		if !m.deliver(message) {
			return
		}
	}
}

func (m *memory[T]) deliver(message T) bool {
	m.subMu.Lock()
	subscribers := append([]*subscriber[T](nil), m.subscribers...)
	m.subMu.Unlock()
	for _, sub := range subscribers {
		if !sub.send(message, m.abort) {
			return false
		}
	}
	return true
}

func (m *memory[T]) shutdown() {
	m.subMu.Lock()
	subscribers := m.subscribers
	m.subscribers = nil
	m.subClosed = true
	m.subMu.Unlock()
	for _, sub := range subscribers {
		sub.close()
	}
}
