package testredis

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func New(t testing.TB) *redis.Client {
	client, _ := NewWithServer(t)
	return client
}

// NewWithServer also returns the miniredis server, whose clock only moves with
// FastForward.
func NewWithServer(t testing.TB) (*redis.Client, *miniredis.Miniredis) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		require.NoError(t, client.Close())
	})
	return client, server
}
