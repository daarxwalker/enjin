package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
)

const (
	googleSendPingTimeout  = 15 * time.Second
	googlePingTimeout      = 10 * time.Second
	googleWriteByteTimeout = 30 * time.Second
	googleTokenEarlyExpiry = 5 * time.Minute
	googleTokenTimeout     = 30 * time.Second
)

func NewAuthConfig(
	clientID,
	clientSecret,
	redirectURL string,
) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		RedirectURL: redirectURL,
		Endpoint:    googleoauth.Endpoint,
	}
}

type SAConfig map[string]string

func NewSAClient(config SAConfig, scopes ...string) (*http.Client, error) {
	if len(scopes) == 0 {
		return nil, errors.New("google scopes are required")
	}
	b, marshalErr := json.Marshal(config)
	if marshalErr != nil {
		return nil, fmt.Errorf("serialize google config failed: %w", marshalErr)
	}
	jwtConfig, jwtErr := googleoauth.JWTConfigFromJSON(b, scopes...)
	if jwtErr != nil {
		return nil, fmt.Errorf("create google jwt config failed: %w", jwtErr)
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.HTTP2 = &http.HTTP2Config{
		SendPingTimeout:  googleSendPingTimeout,
		PingTimeout:      googlePingTimeout,
		WriteByteTimeout: googleWriteByteTimeout,
	}
	source := &googleJWTSource{
		config: jwtConfig,
		client: &http.Client{Transport: transport, Timeout: googleTokenTimeout},
	}
	return &http.Client{
		Transport: &googleTransport{
			base:   transport,
			tokens: &googleTokenCache{source: source},
		},
	}, nil
}

type googleTokenSource interface {
	Token(ctx context.Context) (*oauth2.Token, error)
}

type googleJWTSource struct {
	config *jwt.Config
	client *http.Client
}

func (s *googleJWTSource) Token(ctx context.Context) (*oauth2.Token, error) {
	return s.config.TokenSource(context.WithValue(ctx, oauth2.HTTPClient, s.client)).Token()
}

type googleTokenCache struct {
	source     googleTokenSource
	token      *oauth2.Token
	expiration time.Time
	mu         sync.Mutex
}

func (c *googleTokenCache) current(ctx context.Context) (*oauth2.Token, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != nil && time.Until(c.expiration) > googleTokenEarlyExpiry {
		return c.token, nil
	}
	token, err := c.source.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch google token failed: %w", err)
	}
	c.token = token
	c.expiration = token.Expiry
	return token, nil
}

func (c *googleTokenCache) invalidate(stale *oauth2.Token) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token == stale {
		c.token = nil
	}
}

type googleTransport struct {
	base   http.RoundTripper
	tokens *googleTokenCache
}

func (t *googleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.tokens.current(req.Context())
	if err != nil {
		return nil, err
	}
	res, err := t.base.RoundTrip(t.authorize(req, token))
	if err != nil || res.StatusCode != http.StatusUnauthorized {
		return res, err
	}
	if req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
		return res, nil
	}
	_, _ = io.Copy(io.Discard, res.Body)
	if closeErr := res.Body.Close(); closeErr != nil {
		return nil, fmt.Errorf("close unauthorized google response failed: %w", closeErr)
	}
	t.tokens.invalidate(token)
	token, err = t.tokens.current(req.Context())
	if err != nil {
		return nil, err
	}
	retry := t.authorize(req, token)
	if req.GetBody != nil {
		body, bodyErr := req.GetBody()
		if bodyErr != nil {
			return nil, fmt.Errorf("rewind google request body failed: %w", bodyErr)
		}
		retry.Body = body
	}
	return t.base.RoundTrip(retry)
}

func (t *googleTransport) authorize(req *http.Request, token *oauth2.Token) *http.Request {
	authorized := req.Clone(req.Context())
	token.SetAuthHeader(authorized)
	return authorized
}

func NewSAConfig(
	projectID,
	privateKeyID,
	privateKey,
	clientID,
	clientEmail,
	certURL string,
) SAConfig {
	privateKeyLines := strings.Split(privateKey, "\n")
	privateKeyNormalized := make([]string, 0, len(privateKeyLines))
	for _, line := range privateKeyLines {
		clean := strings.TrimSpace(line)
		privateKeyNormalized = append(privateKeyNormalized, clean)
	}
	privateKey = strings.Join(privateKeyNormalized, "\n")
	return SAConfig{ //nolint:gosec // credentials come from the caller
		"type":                        "service_account",
		"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
		"token_uri":                   "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"universe_domain":             "googleapis.com",
		"project_id":                  projectID,
		"private_key_id":              privateKeyID,
		"private_key":                 privateKey,
		"client_id":                   clientID,
		"client_email":                clientEmail,
		"client_x509_cert_url":        certURL,
	}
}
