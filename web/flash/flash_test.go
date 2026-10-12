package flash_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"testing"

	"github.com/daarxwalker/enjin/client"
	"github.com/daarxwalker/enjin/cookie"
	"github.com/daarxwalker/enjin/infra/cache"
	"github.com/daarxwalker/enjin/internal/testredis"
	"github.com/daarxwalker/enjin/web/flash"

	"github.com/stretchr/testify/require"
)

func TestFlash(t *testing.T) {
	t.Run(
		"post redirect get", func(t *testing.T) {
			c := newTestClient(t)
			c.get("/attach")
			require.Equal(t, flash.Messages{flash.WithSuccess("saved")}, c.read())
			require.Empty(t, c.read())
		},
	)

	t.Run(
		"unread messages wait for a request that reads them", func(t *testing.T) {
			c := newTestClient(t)
			c.get("/attach")
			c.get("/noop")
			c.get("/attach")
			require.Equal(t, flash.Messages{flash.WithSuccess("saved"), flash.WithSuccess("saved")}, c.read())
		},
	)

	t.Run(
		"attach and read in the same request", func(t *testing.T) {
			c := newTestClient(t)
			require.Equal(t, flash.Messages{flash.WithError("failed")}, c.decode(c.get("/both")))
			require.Empty(t, c.read())
		},
	)

	t.Run(
		"messages are scoped to the client", func(t *testing.T) {
			first := newTestClient(t)
			second := first.withNewJar()
			first.get("/attach")
			require.Empty(t, second.read())
			require.Len(t, first.read(), 1)
		},
	)

	t.Run(
		"without middleware", func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			flash.Attach(r.Context(), flash.WithInfo("x"))
			require.Nil(t, flash.From(r.Context()))
		},
	)

	t.Run(
		"without client id", func(t *testing.T) {
			redisClient := testredis.New(t)
			cacheClient := cache.NewClient(cache.NewRedisDriver(redisClient))
			var read flash.Messages
			handler := flash.NewManager(cacheClient).Middleware(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						flash.Attach(r.Context(), flash.WithSuccess("saved"))
						read = flash.From(r.Context())
						w.WriteHeader(http.StatusNoContent)
					},
				),
			)
			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))
			require.Nil(t, read)
			keys, err := redisClient.Keys(t.Context(), "*").Result()
			require.NoError(t, err)
			require.Empty(t, keys)
		},
	)

	t.Run(
		"message types", func(t *testing.T) {
			require.True(t, flash.WithSuccess("x").IsSuccess())
			require.True(t, flash.WithError("x").IsError())
			require.True(t, flash.WithWarning("x").IsWarning())
			require.True(t, flash.WithInfo("x").IsInfo())
			require.True(t, flash.Message{Message: "x"}.IsInfo())
			require.False(t, flash.WithError("x").IsInfo())
		},
	)
}

type testClient struct {
	t    *testing.T
	http *http.Client
	url  string
}

func newTestClient(t *testing.T) *testClient {
	t.Helper()
	cacheClient := cache.NewClient(cache.NewRedisDriver(testredis.New(t)))
	mux := http.NewServeMux()
	mux.HandleFunc(
		"/attach", func(w http.ResponseWriter, r *http.Request) {
			flash.Attach(r.Context(), flash.WithSuccess("saved"))
			w.WriteHeader(http.StatusNoContent)
		},
	)
	mux.HandleFunc(
		"/noop", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	)
	mux.HandleFunc(
		"/read", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(flash.From(r.Context()))
		},
	)
	mux.HandleFunc(
		"/both", func(w http.ResponseWriter, r *http.Request) {
			flash.Attach(r.Context(), flash.WithError("failed"))
			_ = json.NewEncoder(w).Encode(flash.From(r.Context()))
		},
	)
	clientManager := client.NewManager(cacheClient, client.WithConfig(cookie.WithSecure(false)))
	srv := httptest.NewServer(clientManager.Middleware(flash.NewManager(cacheClient).Middleware(mux)))
	t.Cleanup(srv.Close)
	return (&testClient{t: t, url: srv.URL}).withNewJar()
}

func (c *testClient) withNewJar() *testClient {
	c.t.Helper()
	jar, err := cookiejar.New(nil)
	require.NoError(c.t, err)
	return &testClient{t: c.t, http: &http.Client{Jar: jar}, url: c.url}
}

func (c *testClient) get(path string) []byte {
	c.t.Helper()
	res, err := c.http.Get(c.url + path)
	require.NoError(c.t, err)
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	require.NoError(c.t, err)
	return body
}

func (c *testClient) read() flash.Messages {
	c.t.Helper()
	return c.decode(c.get("/read"))
}

func (c *testClient) decode(body []byte) flash.Messages {
	c.t.Helper()
	var messages flash.Messages
	require.NoError(c.t, json.Unmarshal(body, &messages))
	return messages
}
