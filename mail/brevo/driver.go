package brevo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/daarxwalker/enjin/mail"
)

var _ mail.Driver = (*Driver)(nil)

type Driver struct {
	client   *http.Client
	parser   parser
	apiKey   string
	endpoint string
}

const (
	defaultEndpoint  = "https://api.brevo.com/v3/smtp/email"
	requestTimeout   = 30 * time.Second
	maxErrorBodySize = 64 << 10
)

func NewDriver(apiKey string, options ...Option) *Driver {
	d := &Driver{
		client:   &http.Client{Timeout: requestTimeout},
		apiKey:   apiKey,
		endpoint: defaultEndpoint,
	}
	for _, option := range options {
		option(d)
	}
	return d
}

func (d *Driver) Send(ctx context.Context, envelope mail.Envelope, raw []byte) (err error) {
	email, err := d.parser.parse(envelope, raw)
	if err != nil {
		return fmt.Errorf("read brevo email from message failed: %w", err)
	}
	payload, err := json.Marshal(email)
	if err != nil {
		return fmt.Errorf("encode brevo email failed: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create brevo request failed: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", d.apiKey)
	res, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("send brevo email failed: %w", err)
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close brevo response body failed: %w", closeErr))
		}
	}()
	if res.StatusCode == http.StatusCreated || res.StatusCode == http.StatusAccepted {
		return nil
	}
	return d.readError(res)
}

func (d *Driver) readError(res *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(res.Body, maxErrorBodySize))
	if err != nil {
		return fmt.Errorf("brevo responded %s, read body failed: %w", res.Status, err)
	}
	var apiErr errorAPIRow
	if decodeErr := json.Unmarshal(body, &apiErr); decodeErr != nil || apiErr.Message == "" {
		return fmt.Errorf("brevo responded %s: %s", res.Status, bytes.TrimSpace(body))
	}
	return fmt.Errorf("brevo responded %s: %s: %s", res.Status, apiErr.Code, apiErr.Message)
}
