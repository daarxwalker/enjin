package nats

import (
	"fmt"

	natssdk "github.com/nats-io/nats.go"
)

func Connect(uri string) (*natssdk.Conn, error) {
	client, connectBusErr := natssdk.Connect(uri)
	if connectBusErr != nil {
		return nil, fmt.Errorf("bus: connect failed: %w", connectBusErr)
	}
	return client, nil
}
