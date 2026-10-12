package ses

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	sessdk "github.com/aws/aws-sdk-go-v2/service/ses"
)

func Connect(ctx context.Context, region, accessKey, secret string) (*sessdk.Client, error) {
	cfg, createCfgErr := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				accessKey, secret, "",
			),
		),
	)
	if createCfgErr != nil {
		return nil, fmt.Errorf("create email client config failed: %w", createCfgErr)
	}
	return sessdk.NewFromConfig(cfg), nil
}
