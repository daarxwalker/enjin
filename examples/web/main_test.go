package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

var csrfInput = regexp.MustCompile(`name="__csrf__" value="([^"]+)"`)

func TestWeb(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	engine, err := newWeb(context.Background(), rdb, slog.New(slog.DiscardHandler), false)
	require.NoError(t, err)
	srv := httptest.NewServer(engine.Root.Raw())
	t.Cleanup(srv.Close)
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	c := &browser{t: t, http: &http.Client{Jar: jar}, url: srv.URL}

	page := c.home()
	require.Contains(t, page, "<title>Guestbook</title>")
	require.Contains(t, page, `<meta name="description" content="Enjin server-rendered example">`)

	page = c.post(url.Values{"name": {"Alice"}, "message": {"Hi"}})
	require.Contains(t, page, "The form has expired")
	require.NotContains(t, page, "<strong>Alice</strong>")

	page = c.post(url.Values{"__csrf__": {c.token(c.home())}, "name": {"A"}, "message": {""}})
	require.Contains(t, page, "minLen 2")
	require.Contains(t, page, "required")
	require.NotContains(t, c.home(), "minLen 2")

	page = c.post(url.Values{"__csrf__": {c.token(c.home())}, "name": {"Alice"}, "message": {"Hi <there>"}})
	require.Contains(t, page, "Thanks for signing!")
	require.Contains(t, page, "<strong>Alice</strong>: Hi &lt;there&gt;")
	require.NotContains(t, c.home(), "Thanks for signing!")

	res, err := c.http.Get(srv.URL + "/missing")
	require.NoError(t, err)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	require.Equal(t, http.StatusNotFound, res.StatusCode)
	require.Contains(t, string(body), "<h1>Not Found</h1>")
}

type browser struct {
	t    *testing.T
	http *http.Client
	url  string
}

func (b *browser) home() string {
	b.t.Helper()
	res, err := b.http.Get(b.url + "/")
	require.NoError(b.t, err)
	defer func() { _ = res.Body.Close() }()
	return b.read(res)
}

func (b *browser) post(form url.Values) string {
	b.t.Helper()
	res, err := b.http.PostForm(b.url+"/", form)
	require.NoError(b.t, err)
	defer func() { _ = res.Body.Close() }()
	return b.read(res)
}

func (b *browser) read(res *http.Response) string {
	b.t.Helper()
	body, err := io.ReadAll(res.Body)
	require.NoError(b.t, err)
	require.Equal(b.t, http.StatusOK, res.StatusCode)
	return string(body)
}

func (b *browser) token(page string) string {
	b.t.Helper()
	match := csrfInput.FindStringSubmatch(page)
	require.Len(b.t, match, 2)
	return match[1]
}
