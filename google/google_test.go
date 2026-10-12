package google

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

type fakeTokenSource struct {
	lifetime time.Duration
	calls    int
	ctx      context.Context
}

func (s *fakeTokenSource) Token(ctx context.Context) (*oauth2.Token, error) {
	s.ctx = ctx
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.calls++
	return &oauth2.Token{
		AccessToken: "token-" + strconv.Itoa(s.calls),
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(s.lifetime),
	}, nil
}

type recordedRequest struct {
	authorization string
	body          string
}

type fakeGoogleAPI struct {
	rejected map[string]bool
	requests []recordedRequest
}

func (a *fakeGoogleAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	authorization := r.Header.Get("Authorization")
	a.requests = append(a.requests, recordedRequest{authorization: authorization, body: string(body)})
	if a.rejected[authorization] {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func newGoogleTestClient(source *fakeTokenSource) *http.Client {
	return &http.Client{
		Transport: &googleTransport{
			base:   http.DefaultTransport,
			tokens: &googleTokenCache{source: source},
		},
	}
}

func doGoogleTestRequest(t *testing.T, client *http.Client, method, url, body string) int {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request failed: %v", err)
	}
	_ = res.Body.Close()
	return res.StatusCode
}

func TestGoogleTransport_ReusesValidToken(t *testing.T) {
	api := &fakeGoogleAPI{}
	server := httptest.NewServer(api)
	defer server.Close()
	source := &fakeTokenSource{lifetime: time.Hour}
	client := newGoogleTestClient(source)

	doGoogleTestRequest(t, client, http.MethodGet, server.URL, "")
	doGoogleTestRequest(t, client, http.MethodGet, server.URL, "")

	if source.calls != 1 {
		t.Fatalf("expected 1 token fetch, got %d", source.calls)
	}
	for _, request := range api.requests {
		if request.authorization != "Bearer token-1" {
			t.Fatalf("expected Bearer token-1, got %q", request.authorization)
		}
	}
}

func TestGoogleTransport_RefreshesTokenBeforeExpiry(t *testing.T) {
	api := &fakeGoogleAPI{}
	server := httptest.NewServer(api)
	defer server.Close()
	source := &fakeTokenSource{lifetime: googleTokenEarlyExpiry - time.Minute}
	client := newGoogleTestClient(source)

	doGoogleTestRequest(t, client, http.MethodGet, server.URL, "")
	doGoogleTestRequest(t, client, http.MethodGet, server.URL, "")

	if source.calls != 2 {
		t.Fatalf("expected 2 token fetches, got %d", source.calls)
	}
	if api.requests[1].authorization != "Bearer token-2" {
		t.Fatalf("expected Bearer token-2, got %q", api.requests[1].authorization)
	}
}

func TestGoogleTransport_RetriesWithNewTokenOnUnauthorized(t *testing.T) {
	api := &fakeGoogleAPI{rejected: map[string]bool{"Bearer token-1": true}}
	server := httptest.NewServer(api)
	defer server.Close()
	source := &fakeTokenSource{lifetime: time.Hour}
	client := newGoogleTestClient(source)

	status := doGoogleTestRequest(t, client, http.MethodPost, server.URL, "payload")

	if status != http.StatusOK {
		t.Fatalf("expected 200, got %d", status)
	}
	if len(api.requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(api.requests))
	}
	retried := api.requests[1]
	if retried.authorization != "Bearer token-2" {
		t.Fatalf("expected Bearer token-2, got %q", retried.authorization)
	}
	if retried.body != "payload" {
		t.Fatalf("expected retried body payload, got %q", retried.body)
	}
}

func TestGoogleTransport_RetriesOnlyOnce(t *testing.T) {
	api := &fakeGoogleAPI{rejected: map[string]bool{"Bearer token-1": true, "Bearer token-2": true}}
	server := httptest.NewServer(api)
	defer server.Close()
	source := &fakeTokenSource{lifetime: time.Hour}
	client := newGoogleTestClient(source)

	status := doGoogleTestRequest(t, client, http.MethodGet, server.URL, "")

	if status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", status)
	}
	if len(api.requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(api.requests))
	}
}

func TestGoogleTransport_FetchesTokenWithRequestContext(t *testing.T) {
	api := &fakeGoogleAPI{}
	server := httptest.NewServer(api)
	defer server.Close()
	source := &fakeTokenSource{lifetime: time.Hour}
	client := newGoogleTestClient(source)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}

	res, err := client.Do(req)

	if res != nil {
		_ = res.Body.Close()
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
	if source.ctx != ctx {
		t.Fatalf("expected token fetch with request context")
	}
	if len(api.requests) != 0 {
		t.Fatalf("expected no requests, got %d", len(api.requests))
	}
}

func TestNewSAClient_RequiresScopes(t *testing.T) {
	_, err := NewSAClient(SAConfig{})

	if err == nil {
		t.Fatalf("expected error without scopes")
	}
}
