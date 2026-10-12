package ses

import (
	"context"
	"fmt"

	"github.com/daarxwalker/enjin/mail"

	"github.com/aws/aws-sdk-go-v2/aws"
	sessdk "github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

var _ mail.Driver = (*Driver)(nil)

type Driver struct {
	client *sessdk.Client
}

func NewDriver(client *sessdk.Client) *Driver {
	d := &Driver{
		client: client,
	}
	return d
}

func (d *Driver) Send(ctx context.Context, envelope mail.Envelope, raw []byte) error {
	_, sendErr := d.client.SendRawEmail(
		ctx,
		&sessdk.SendRawEmailInput{
			Source:       aws.String(envelope.From),
			Destinations: envelope.Recipients,
			RawMessage:   &types.RawMessage{Data: raw},
		},
	)
	if sendErr != nil {
		return fmt.Errorf("send aws ses raw email failed: %w", sendErr)
	}
	return nil
}
