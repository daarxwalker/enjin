package enjintest

import (
	"context"
	"strconv"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	testvalkey "github.com/testcontainers/testcontainers-go/modules/valkey"
	"github.com/testcontainers/testcontainers-go/wait"
)

func NewValkey(ctx context.Context, t testing.TB, options ...Option) *redis.Client {
	skipShort(t)
	cfg := config{
		version: "9.1.0-alpine",
		port:    6379,
	}
	for _, opt := range options {
		opt(&cfg)
	}
	container, runErr := testvalkey.Run(
		ctx,
		"docker.io/valkey/valkey:"+cfg.version,
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort(strconv.Itoa(cfg.port)+"/tcp"),
		),
	)
	require.NoError(t, runErr)

	endpoint, err := container.Endpoint(ctx, "")
	require.NoError(t, err)

	t.Cleanup(
		func() {
			require.NoError(t, container.Terminate(context.WithoutCancel(ctx)))
			t.Log("cleanup cache done")
		},
	)

	client := redis.NewClient(&redis.Options{
		Addr: endpoint,
	})
	require.NoError(t, client.Ping(ctx).Err())

	return client
}
