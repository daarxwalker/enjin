package httpx

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const maxErrorBody = 64 << 10

type Client struct {
	raw *http.Client
}

func NewClient(options ...Option) *Client {
	cfg := &clientConfig{
		timeout:               5 * time.Minute,
		DialTimeout:           3 * time.Second,
		keepAlive:             30 * time.Second,
		forceHTTP2:            true,
		maxIdleConns:          100,
		maxIdleConnsPerHost:   20,
		idleConnTimeout:       90 * time.Second,
		tlsHandshakeTimeout:   3 * time.Second,
		responseHeaderTimeout: 2 * time.Second,
	}
	for _, opt := range options {
		opt(cfg)
	}
	return &Client{
		raw: &http.Client{
			Timeout: cfg.timeout,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   cfg.DialTimeout,
					KeepAlive: cfg.keepAlive,
				}).DialContext,
				ForceAttemptHTTP2:     cfg.forceHTTP2,
				MaxIdleConns:          cfg.maxIdleConns,
				MaxIdleConnsPerHost:   cfg.maxIdleConnsPerHost,
				IdleConnTimeout:       cfg.idleConnTimeout,
				TLSHandshakeTimeout:   cfg.tlsHandshakeTimeout,
				ResponseHeaderTimeout: cfg.responseHeaderTimeout,
			},
		},
	}
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	res, err := c.raw.Do(req) //nolint:gosec // the caller owns the request URL
	if err != nil {
		return nil, fmt.Errorf("do request failed: %w", err)
	}
	if res.StatusCode >= http.StatusBadRequest {
		bts, readErr := io.ReadAll(io.LimitReader(res.Body, maxErrorBody))
		closeErr := res.Body.Close()
		res.Body = http.NoBody
		if readErr != nil {
			return res, errors.Join(fmt.Errorf("read error response failed: %w", readErr), closeErr)
		}
		return res, errors.Join(fmt.Errorf("error response: %s", string(bts)), closeErr)
	}
	return res, nil
}

type clientConfig struct {
	timeout               time.Duration
	DialTimeout           time.Duration
	keepAlive             time.Duration
	idleConnTimeout       time.Duration
	tlsHandshakeTimeout   time.Duration
	responseHeaderTimeout time.Duration
	maxIdleConns          int
	maxIdleConnsPerHost   int
	forceHTTP2            bool
}

type Option func(*clientConfig)

func WithClientTimeout(timeout time.Duration) Option {
	return func(c *clientConfig) {
		c.timeout = timeout
	}
}

func WithClientDialTimeout(dialTimeout time.Duration) Option {
	return func(c *clientConfig) {
		c.DialTimeout = dialTimeout
	}
}

func WithClientKeepAlive(keepAlive time.Duration) Option {
	return func(c *clientConfig) {
		c.keepAlive = keepAlive
	}
}

func WithClientForceHTTP2(forceHTTP2 bool) Option {
	return func(c *clientConfig) {
		c.forceHTTP2 = forceHTTP2
	}
}

func WithClientMaxIdleConns(maxIdleConns int) Option {
	return func(c *clientConfig) {
		c.maxIdleConns = maxIdleConns
	}
}

func WithClientMaxIdleConnsPerHost(maxIdleConnsPerHost int) Option {
	return func(c *clientConfig) {
		c.maxIdleConnsPerHost = maxIdleConnsPerHost
	}
}

func WithClientIdleConnTimeout(idleConnTimeout time.Duration) Option {
	return func(c *clientConfig) {
		c.idleConnTimeout = idleConnTimeout
	}
}

func WithClientTLSHandshakeTimeout(tlsHandshakeTimeout time.Duration) Option {
	return func(c *clientConfig) {
		c.tlsHandshakeTimeout = tlsHandshakeTimeout
	}
}

func WithClientResponseHeaderTimeout(responseHeaderTimeout time.Duration) Option {
	return func(c *clientConfig) {
		c.responseHeaderTimeout = responseHeaderTimeout
	}
}
