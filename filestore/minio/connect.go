package minio

import (
	"fmt"

	miniosdk "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func Connect(endpoint, accessKey, secretKey string) (*miniosdk.Client, error) {
	client, err := miniosdk.New(endpoint, &miniosdk.Options{
		Creds: credentials.NewStaticV4(accessKey, secretKey, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("connect minio to %s failed: %w", endpoint, err)
	}
	return client, nil
}
