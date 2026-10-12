package bus

import "sync"

type subscriber[T any] struct {
	ch        chan T
	done      chan struct{}
	closeOnce sync.Once
	mu        sync.Mutex
	closed    bool
}

func newSubscriber[T any](bufSize int) *subscriber[T] {
	return &subscriber[T]{
		ch:   make(chan T, bufSize),
		done: make(chan struct{}),
	}
}

func (s *subscriber[T]) send(message T, abort <-chan struct{}) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return true
	}
	select {
	case s.ch <- message:
		return true
	case <-s.done:
		return true
	case <-abort:
		return false
	}
}

func (s *subscriber[T]) close() {
	s.closeOnce.Do(func() {
		close(s.done)
		s.mu.Lock()
		s.closed = true
		close(s.ch)
		s.mu.Unlock()
	})
}
