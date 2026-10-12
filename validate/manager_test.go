package validate_test

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
	"github.com/daarxwalker/enjin/validate"

	"github.com/stretchr/testify/require"
)

func TestManager(t *testing.T) {
	t.Run(
		"post redirect get", func(t *testing.T) {
			c := newTestClient(t)
			c.get("/attach")
			require.Equal(t, validate.Result{"name": {validate.Required}}, c.read())
			require.Nil(t, c.read())
		},
	)

	t.Run(
		"results are scoped to the client", func(t *testing.T) {
			first := newTestClient(t)
			second := first.withNewJar()
			first.get("/attach")
			require.Nil(t, second.read())
			require.NotNil(t, first.read())
		},
	)

	t.Run(
		"without client id", func(t *testing.T) {
			redisClient := testredis.New(t)
			cacheClient := cache.NewClient(cache.NewRedisDriver(redisClient))
			var read validate.Result
			handler := validate.NewManager(cacheClient).Middleware(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						validate.Attach(r.Context(), validate.Result{"name": {validate.Required}})
						read = validate.From(r.Context())
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
		"without middleware", func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			validate.Attach(r.Context(), validate.Result{"name": {validate.Required}})
			require.Nil(t, validate.From(r.Context()))
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
			validate.Attach(r.Context(), validate.Result{"name": {validate.Required}})
			w.WriteHeader(http.StatusNoContent)
		},
	)
	mux.HandleFunc(
		"/read", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(validate.From(r.Context()))
		},
	)
	clientManager := client.NewManager(cacheClient, client.WithConfig(cookie.WithSecure(false)))
	srv := httptest.NewServer(clientManager.Middleware(validate.NewManager(cacheClient).Middleware(mux)))
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

func (c *testClient) read() validate.Result {
	c.t.Helper()
	var result validate.Result
	require.NoError(c.t, json.Unmarshal(c.get("/read"), &result))
	return result
}
