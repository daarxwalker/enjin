package bus

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeDriver struct {
	subscribeErr error
	handler      func(ctx context.Context, data []byte) error
	published    [][]byte
	mu           sync.Mutex
	closed       bool
}

func (d *fakeDriver) Publish(_ context.Context, _ string, data []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.published = append(d.published, data)
	return nil
}

func (d *fakeDriver) TryPublish(subject string, data []byte) error {
	return d.Publish(context.Background(), subject, data)
}

func (d *fakeDriver) Subscribe(
	_, _ string,
	handler func(ctx context.Context, data []byte) error,
) (Subscription, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.subscribeErr != nil {
		return nil, d.subscribeErr
	}
	d.handler = handler
	return d, nil
}

func (d *fakeDriver) Close(context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	return nil
}

type note struct {
	Text string `json:"text"`
}

func receiveBound(t *testing.T, ch <-chan note) note {
	t.Helper()
	select {
	case message, ok := <-ch:
		require.True(t, ok, "channel closed")
		return message
	case <-time.After(time.Second):
		t.Fatal("no message received")
	}
	return note{}
}

func TestUse(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() {
		binding.mu.Lock()
		defer binding.mu.Unlock()
		binding.driver = nil
		binding.resolved = false
	})
	driver := &fakeDriver{}
	remoteEvent := NewEvent[note](4, WithSubject("note"), WithQueue("notes"))
	localEvent := NewEvent[note](4)
	require.NoError(t, Use(driver))

	require.NoError(t, remoteEvent.Publish(ctx, note{Text: "hello"}))
	require.NoError(t, remoteEvent.TryPublish(note{Text: "again"}))
	require.Equal(t, [][]byte{[]byte(`{"text":"hello"}`), []byte(`{"text":"again"}`)}, driver.published)

	sub, err := remoteEvent.Subscribe(1)
	require.NoError(t, err)
	require.NoError(t, driver.handler(ctx, []byte(`{"text":"incoming"}`)))
	require.Equal(t, note{Text: "incoming"}, receiveBound(t, sub))
	require.Error(t, driver.handler(ctx, []byte(`{`)))

	localSub, err := localEvent.Subscribe(1)
	require.NoError(t, err)
	require.NoError(t, localEvent.Publish(ctx, note{Text: "local"}))
	require.Equal(t, note{Text: "local"}, receiveBound(t, localSub))
	require.Len(t, driver.published, 2)

	require.ErrorIs(t, Use(driver), ErrInUse)

	require.NoError(t, remoteEvent.Close(ctx))
	require.NoError(t, localEvent.Close(ctx))
	require.True(t, driver.closed)
	_, open := <-sub
	require.False(t, open)
	require.ErrorIs(t, remoteEvent.Publish(ctx, note{}), ErrClosed)
}

func TestRemoteSubscribeFailure(t *testing.T) {
	driverErr := errors.New("subscribe failed")
	driver := &fakeDriver{subscribeErr: driverErr}
	remoteEvent := newRemote[note](driver, route{subject: "note"}, 4)
	sub, err := remoteEvent.Subscribe(1)
	require.ErrorIs(t, err, driverErr)
	_, open := <-sub
	require.False(t, open)
	driver.mu.Lock()
	driver.subscribeErr = nil
	driver.mu.Unlock()
	_, err = remoteEvent.Subscribe(1)
	require.NoError(t, err)
	require.NoError(t, remoteEvent.Close(context.Background()))
}
