package brevo

import "net/http"

type Option func(*Driver)

func WithHTTPClient(client *http.Client) Option {
	return func(d *Driver) {
		if client != nil {
			d.client = client
		}
	}
}

func WithEndpoint(endpoint string) Option {
	return func(d *Driver) {
		d.endpoint = endpoint
	}
}
