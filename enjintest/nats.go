package enjintest

import (
	"context"
	"strconv"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func NewNATS(ctx context.Context, t testing.TB, options ...Option) *nats.Conn {
	skipShort(t)
	cfg := config{
		version: "2.12.4-alpine",
		port:    4222,
	}
	for _, opt := range options {
		opt(&cfg)
	}
	port := strconv.Itoa(cfg.port) + "/tcp"
	container, runErr := testcontainers.Run(
		ctx,
		"docker.io/library/nats:"+cfg.version,
		testcontainers.WithExposedPorts(port),
		testcontainers.WithCmd("-js", "-p", strconv.Itoa(cfg.port)),
		testcontainers.WithWaitStrategy(wait.ForListeningPort(port)),
	)
	require.NoError(t, runErr)

	endpoint, err := container.PortEndpoint(ctx, port, "nats")
	require.NoError(t, err)

	conn, err := nats.Connect(endpoint)
	require.NoError(t, err)

	t.Cleanup(
		func() {
			conn.Close()
			require.NoError(t, container.Terminate(context.WithoutCancel(ctx)))
			t.Log("cleanup bus done")
		},
	)

	return conn
}
