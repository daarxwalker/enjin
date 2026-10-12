package nats

import (
	"context"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/bus"
	"github.com/daarxwalker/enjin/enjintest"

	"github.com/stretchr/testify/require"
)

const wait = 5 * time.Second

func subscribe(t *testing.T, d *Driver, subject, queue string) (<-chan string, bus.Subscription) {
	t.Helper()
	ch := make(chan string, 16)
	sub, err := d.Subscribe(subject, queue, func(_ context.Context, data []byte) error {
		ch <- string(data)
		return nil
	})
	require.NoError(t, err)
	return ch, sub
}

func receive(t *testing.T, ch <-chan string) string {
	t.Helper()
	select {
	case message := <-ch:
		return message
	case <-time.After(wait):
		t.Fatal("no message received")
	}
	return ""
}

func count(chs ...<-chan string) int {
	var total int
	for _, ch := range chs {
		for open := true; open; {
			select {
			case <-ch:
				total++
			case <-time.After(200 * time.Millisecond):
				open = false
			}
		}
	}
	return total
}

func closeSubscription(t *testing.T, sub bus.Subscription) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), wait)
	defer cancel()
	require.NoError(t, sub.Close(ctx))
}

func TestDriver(t *testing.T) {
	conn := enjintest.NewNATS(t.Context(), t)

	core, err := NewDriver(t.Context(), conn, WithPrefix("core"))
	require.NoError(t, err)
	js, err := NewDriver(t.Context(), conn, WithPrefix("js"), WithJetStream("JS", time.Minute))
	require.NoError(t, err)

	t.Run(
		"core broadcast to every instance", func(t *testing.T) {
			first, firstSub := subscribe(t, core, "broadcast", "")
			second, secondSub := subscribe(t, core, "broadcast", "")
			defer closeSubscription(t, firstSub)
			defer closeSubscription(t, secondSub)
			require.NoError(t, conn.Flush())
			require.NoError(t, core.Publish(t.Context(), "broadcast", []byte("hello")))
			require.Equal(t, "hello", receive(t, first))
			require.Equal(t, "hello", receive(t, second))
		},
	)

	t.Run(
		"core queue delivers once", func(t *testing.T) {
			first, firstSub := subscribe(t, core, "queue", "workers")
			second, secondSub := subscribe(t, core, "queue", "workers")
			defer closeSubscription(t, firstSub)
			defer closeSubscription(t, secondSub)
			require.NoError(t, conn.Flush())
			for range 10 {
				require.NoError(t, core.TryPublish("queue", []byte("job")))
			}
			require.Equal(t, 10, count(first, second))
		},
	)

	t.Run(
		"core close drains pending messages", func(t *testing.T) {
			ch, sub := subscribe(t, core, "drain", "")
			require.NoError(t, conn.Flush())
			for range 3 {
				require.NoError(t, core.Publish(t.Context(), "drain", []byte("item")))
			}
			require.NoError(t, conn.Flush())
			closeSubscription(t, sub)
			require.Len(t, ch, 3)
		},
	)

	t.Run(
		"jetstream broadcast to every instance", func(t *testing.T) {
			first, firstSub := subscribe(t, js, "broadcast", "")
			second, secondSub := subscribe(t, js, "broadcast", "")
			defer closeSubscription(t, firstSub)
			defer closeSubscription(t, secondSub)
			require.NoError(t, js.Publish(t.Context(), "broadcast", []byte("hello")))
			require.Equal(t, "hello", receive(t, first))
			require.Equal(t, "hello", receive(t, second))
		},
	)

	t.Run(
		"jetstream queue delivers once", func(t *testing.T) {
			first, firstSub := subscribe(t, js, "queue", "workers")
			second, secondSub := subscribe(t, js, "queue", "workers")
			defer closeSubscription(t, firstSub)
			defer closeSubscription(t, secondSub)
			for range 10 {
				require.NoError(t, js.Publish(t.Context(), "queue", []byte("job")))
			}
			require.Equal(t, 10, count(first, second))
		},
	)

	t.Run(
		"jetstream queue keeps messages while offline", func(t *testing.T) {
			_, offline := subscribe(t, js, "durable", "mailer")
			closeSubscription(t, offline)
			require.NoError(t, js.Publish(t.Context(), "durable", []byte("mail")))
			online, onlineSub := subscribe(t, js, "durable", "mailer")
			defer closeSubscription(t, onlineSub)
			require.Equal(t, "mail", receive(t, online))
		},
	)
}
