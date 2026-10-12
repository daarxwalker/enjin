package enjintest

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	filestoreAccessKey = "enjin"
	filestoreSecretKey = "enjin-secret"
)

func NewSeaweedFS(ctx context.Context, t testing.TB, bucket string, options ...Option) *minio.Client {
	skipShort(t)
	cfg := config{
		version: "4.05",
		port:    8333,
	}
	for _, opt := range options {
		opt(&cfg)
	}
	port := strconv.Itoa(cfg.port)
	container, runErr := testcontainers.Run(
		ctx,
		"docker.io/chrislusf/seaweedfs:"+cfg.version,
		testcontainers.WithExposedPorts(port+"/tcp"),
		testcontainers.WithEnv(map[string]string{
			"AWS_ACCESS_KEY_ID":     filestoreAccessKey,
			"AWS_SECRET_ACCESS_KEY": filestoreSecretKey,
		}),
		testcontainers.WithCmd("server", "-s3", "-s3.port="+port, "-dir=/data"),
		testcontainers.WithWaitStrategy(
			wait.ForHTTP("/healthz").WithPort(port+"/tcp").WithStartupTimeout(2*time.Minute),
		),
	)
	require.NoError(t, runErr)

	endpoint, err := container.PortEndpoint(ctx, port+"/tcp", "")
	require.NoError(t, err)

	t.Cleanup(
		func() {
			require.NoError(t, container.Terminate(context.WithoutCancel(ctx)))
			t.Log("cleanup filestore done")
		},
	)

	client, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(filestoreAccessKey, filestoreSecretKey, ""),
	})
	require.NoError(t, err)
	require.NoError(t, client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}))

	return client
}
