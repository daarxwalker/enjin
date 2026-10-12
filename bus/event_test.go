package bus_test

import (
	"context"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/bus"

	"github.com/stretchr/testify/require"
)

const wait = time.Second

func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case message, ok := <-ch:
		require.True(t, ok, "channel closed")
		return message
	case <-time.After(wait):
		t.Fatal("no message received")
	}
	var zero T
	return zero
}

func requireSubscribe[T any](t *testing.T, e *bus.Event[T], bufSize int) <-chan T {
	t.Helper()
	sub, err := e.Subscribe(bufSize)
	require.NoError(t, err)
	return sub
}

func requireClosed[T any](t *testing.T, ch <-chan T) {
	t.Helper()
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-time.After(wait):
			t.Fatal("channel not closed")
		}
	}
}

func requireDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(wait):
		t.Fatal("blocked")
	}
}

func closeEvent[T any](ctx context.Context, t *testing.T, e *bus.Event[T]) {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	require.NoError(t, e.Close(ctx))
}

func TestEvent(t *testing.T) {
	ctx := context.Background()

	t.Run(
		"deliver to all subscribers in order", func(t *testing.T) {
			e := bus.NewEvent[int](4)
			defer closeEvent(ctx, t, e)
			first, second := requireSubscribe(t, e, 4), requireSubscribe(t, e, 4)
			for i := range 3 {
				require.NoError(t, e.Publish(ctx, i))
			}
			for i := range 3 {
				require.Equal(t, i, receive(t, first))
				require.Equal(t, i, receive(t, second))
			}
		},
	)

	t.Run(
		"publish after close", func(t *testing.T) {
			e := bus.NewEvent[int](1)
			closeEvent(ctx, t, e)
			require.ErrorIs(t, e.Publish(ctx, 1), bus.ErrClosed)
			require.ErrorIs(t, e.TryPublish(1), bus.ErrClosed)
		},
	)

	t.Run(
		"close delivers buffered messages", func(t *testing.T) {
			e := bus.NewEvent[int](64)
			sub := requireSubscribe(t, e, 0)
			received := make(chan int, 20)
			go func() {
				defer close(received)
				for message := range sub {
					received <- message
				}
			}()
			for i := range 20 {
				require.NoError(t, e.Publish(ctx, i))
			}
			closeEvent(ctx, t, e)
			var got []int
			for message := range received {
				got = append(got, message)
			}
			require.Len(t, got, 20)
		},
	)

	t.Run(
		"close closes subscribers before return", func(t *testing.T) {
			e := bus.NewEvent[int](1)
			sub := requireSubscribe(t, e, 1)
			closeEvent(ctx, t, e)
			select {
			case _, ok := <-sub:
				require.False(t, ok)
			default:
				t.Fatal("subscriber open after close")
			}
		},
	)

	t.Run(
		"close gives up on stuck subscriber", func(t *testing.T) {
			e := bus.NewEvent[int](4)
			sub := requireSubscribe(t, e, 0)
			require.NoError(t, e.Publish(ctx, 1))
			require.NoError(t, e.Publish(ctx, 2))
			closeCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			require.ErrorIs(t, e.Close(closeCtx), context.DeadlineExceeded)
			requireClosed(t, sub)
		},
	)

	t.Run(
		"subscribe after close", func(t *testing.T) {
			e := bus.NewEvent[int](1)
			closeEvent(ctx, t, e)
			sub, err := e.Subscribe(1)
			require.ErrorIs(t, err, bus.ErrClosed)
			requireClosed(t, sub)
		},
	)

	t.Run(
		"unsubscribe stuck subscriber", func(t *testing.T) {
			e := bus.NewEvent[int](4)
			defer closeEvent(ctx, t, e)
			other := requireSubscribe(t, e, 4)
			stuck := requireSubscribe(t, e, 0)
			require.NoError(t, e.Publish(ctx, 1))
			require.Equal(t, 1, receive(t, other))
			done := make(chan struct{})
			go func() {
				e.Unsubscribe(stuck)
				close(done)
			}()
			requireDone(t, done)
			requireClosed(t, stuck)
			require.NoError(t, e.Publish(ctx, 2))
			require.Equal(t, 2, receive(t, other))
		},
	)

	t.Run(
		"subscribe while dispatcher is blocked", func(t *testing.T) {
			e := bus.NewEvent[int](4)
			stuck := requireSubscribe(t, e, 0)
			require.NoError(t, e.Publish(ctx, 1))
			done := make(chan struct{})
			go func() {
				_, _ = e.Subscribe(1)
				close(done)
			}()
			requireDone(t, done)
			e.Unsubscribe(stuck)
			closeEvent(ctx, t, e)
		},
	)

	t.Run(
		"unsubscribe unknown channel", func(t *testing.T) {
			e := bus.NewEvent[int](1)
			defer closeEvent(ctx, t, e)
			e.Unsubscribe(make(chan int))
		},
	)

	t.Run(
		"publish waits for space", func(t *testing.T) {
			e := bus.NewEvent[int](1)
			stuck := requireSubscribe(t, e, 0)
			require.NoError(t, e.Publish(ctx, 1))
			publishCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			require.NoError(t, e.Publish(publishCtx, 2))
			require.ErrorIs(t, e.Publish(publishCtx, 3), context.DeadlineExceeded)
			require.ErrorIs(t, e.TryPublish(3), bus.ErrFull)
			require.Equal(t, 1, receive(t, stuck))
			require.Equal(t, 2, receive(t, stuck))
			closeEvent(ctx, t, e)
		},
	)

	t.Run(
		"close releases blocked publish", func(t *testing.T) {
			e := bus.NewEvent[int](1)
			stuck := requireSubscribe(t, e, 0)
			require.NoError(t, e.Publish(ctx, 1))
			require.NoError(t, e.Publish(ctx, 2))
			published := make(chan error, 1)
			go func() {
				published <- e.Publish(ctx, 3)
			}()
			closeCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			_ = e.Close(closeCtx)
			require.ErrorIs(t, <-published, bus.ErrClosed)
			requireClosed(t, stuck)
		},
	)
}
