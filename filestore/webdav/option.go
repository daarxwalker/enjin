package webdav

import (
	"encoding/base64"
	"net/http"
)

type Option func(*Driver)

func WithBasicAuth(username, password string) Option {
	return func(d *Driver) {
		d.authorization = "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
	}
}

func WithBearerToken(token string) Option {
	return func(d *Driver) {
		d.authorization = "Bearer " + token
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(d *Driver) {
		d.client = client
	}
}
