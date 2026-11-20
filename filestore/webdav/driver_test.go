package webdav_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/filestore/filestoretest"
	"github.com/daarxwalker/enjin/filestore/webdav"

	"github.com/stretchr/testify/require"
	xwebdav "golang.org/x/net/webdav"
)

func TestDriver(t *testing.T) {
	server := newServer(t, nil)
	filestoretest.RunDriverSuite(t, filestoretest.Subject{
		Driver:             newDriver(t, server.URL+"/dav"),
		DerivesContentType: true,
	})
}

func TestNewDriver(t *testing.T) {
	for _, baseURL := range []string{"ftp://example.com/dav/", "http:///dav/", "http://example.com/dav/?a=1", "::"} {
		_, err := webdav.NewDriver(baseURL)
		require.Error(t, err, baseURL)
	}
}

func TestDriverAuth(t *testing.T) {
	ctx := context.Background()

	t.Run("basic", func(t *testing.T) {
		server := newServer(t, func(r *http.Request) bool {
			username, password, ok := r.BasicAuth()
			return ok && username == "app" && password == "secret"
		})
		anonymous := newDriver(t, server.URL+"/dav/")
		require.ErrorIs(t, putText(ctx, anonymous, "note.txt", "x"), webdav.ErrUnexpectedStatus)

		signedIn := newDriver(t, server.URL+"/dav/", webdav.WithBasicAuth("app", "secret"))
		require.NoError(t, putText(ctx, signedIn, "note.txt", "x"))
	})

	t.Run("bearer", func(t *testing.T) {
		server := newServer(t, func(r *http.Request) bool {
			return r.Header.Get("Authorization") == "Bearer token"
		})
		anonymous := newDriver(t, server.URL+"/dav/")
		_, _, err := anonymous.Stat(ctx, "note.txt")
		require.ErrorIs(t, err, webdav.ErrUnexpectedStatus)

		signedIn := newDriver(t, server.URL+"/dav/", webdav.WithBearerToken("token"))
		require.NoError(t, putText(ctx, signedIn, "note.txt", "x"))
	})
}

func TestDriverKeys(t *testing.T) {
	ctx := context.Background()
	var requests atomic.Int32
	server := newServer(t, func(*http.Request) bool {
		requests.Add(1)
		return true
	})
	driver := newDriver(t, server.URL+"/dav/")
	for _, key := range []string{"", ".", "/abs.txt", "../up.txt", "a/../b.txt", "a//b.txt", "a/", `a\b.txt`} {
		require.ErrorIs(t, putText(ctx, driver, key, "x"), webdav.ErrInvalidKey, key)
		_, _, err := driver.Get(ctx, key)
		require.ErrorIs(t, err, webdav.ErrInvalidKey, key)
		_, _, err = driver.Stat(ctx, key)
		require.ErrorIs(t, err, webdav.ErrInvalidKey, key)
		require.ErrorIs(t, driver.Delete(ctx, key), webdav.ErrInvalidKey, key)
	}
	require.Zero(t, requests.Load())
}

func TestDriverCollections(t *testing.T) {
	ctx := context.Background()
	server := newServer(t, nil)
	driver := newDriver(t, server.URL+"/dav/")

	t.Run("put recreates removed collections", func(t *testing.T) {
		require.NoError(t, putText(ctx, driver, "tree/deep/a.txt", "a"))
		deleteOnServer(ctx, t, server.URL+"/dav/tree/")

		require.NoError(t, putText(ctx, driver, "tree/deep/b.txt", "b"))
		requireContent(ctx, t, driver, "tree/deep/b.txt", "b")
	})

	t.Run("a body that cannot seek fails once", func(t *testing.T) {
		require.NoError(t, putText(ctx, driver, "once/a.txt", "a"))
		deleteOnServer(ctx, t, server.URL+"/dav/once/")

		body := io.MultiReader(strings.NewReader("b"))
		require.ErrorIs(t, driver.Put(ctx, "once/b.txt", body, 1, "text/plain"), webdav.ErrUnexpectedStatus)
		require.NoError(t, putText(ctx, driver, "once/b.txt", "b"))
	})

	t.Run("delete of a collection key deletes nothing", func(t *testing.T) {
		require.NoError(t, putText(ctx, driver, "kept/a.txt", "a"))

		require.NoError(t, driver.Delete(ctx, "kept"))
		requireContent(ctx, t, driver, "kept/a.txt", "a")
	})

	t.Run("put leaves the body open", func(t *testing.T) {
		body := &closeRecorder{Reader: strings.NewReader("open")}
		require.NoError(t, driver.Put(ctx, "open.txt", body, 4, "text/plain"))
		require.False(t, body.closed)
	})
}

func TestDriverForeignAnswers(t *testing.T) {
	ctx := context.Background()
	var host string
	mux := http.NewServeMux()
	mux.HandleFunc("/remote.php/dav/files/app/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PROPFIND" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		answer, ok := foreignAnswers[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.WriteHeader(http.StatusMultiStatus)
		_, err := io.WriteString(w, strings.ReplaceAll(answer, "{host}", host))
		if err != nil {
			t.Errorf("write foreign answer failed: %v", err)
		}
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	host = server.URL
	driver := newDriver(t, server.URL+"/remote.php/dav/files/app")

	files, err := driver.List(ctx, "")
	require.NoError(t, err)
	require.Len(t, files, 2)
	require.Equal(t, "manual.pdf", files[0].Key)
	require.Equal(t, "application/pdf", files[0].ContentType)
	require.Equal(t, "def456", files[0].ETag)
	require.Equal(t, "photos/kůň #1.jpg", files[1].Key)
	require.Equal(t, "image/jpeg", files[1].ContentType)
	require.Equal(t, "abc123", files[1].ETag)
	require.Equal(t, int64(2048), files[1].Size)
	require.Equal(t, time.Date(2026, time.November, 20, 10, 0, 0, 0, time.UTC), files[1].ModifiedAt.UTC())

	file, exists, err := driver.Stat(ctx, "manual.pdf")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, int64(12), file.Size)

	_, exists, err = driver.Stat(ctx, "photos")
	require.NoError(t, err)
	require.False(t, exists)
}

// foreignAnswers imitate Nextcloud: absolute hrefs, percent-encoded names,
// weak ETags, a file without getcontenttype and a 404 propstat for the
// properties a collection does not have.
var foreignAnswers = map[string]string{
	"/remote.php/dav/files/app/": `<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:">
 <d:response>
  <d:href>/remote.php/dav/files/app/</d:href>
  <d:propstat><d:prop><d:resourcetype><d:collection/></d:resourcetype></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat>
 </d:response>
 <d:response>
  <d:href>{host}/remote.php/dav/files/app/photos/</d:href>
  <d:propstat><d:prop><d:resourcetype><d:collection/></d:resourcetype></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat>
  <d:propstat><d:prop><d:getcontentlength/></d:prop><d:status>HTTP/1.1 404 Not Found</d:status></d:propstat>
 </d:response>
 <d:response>
  <d:href>/remote.php/dav/files/app/manual.pdf</d:href>
  <d:propstat><d:prop>
   <d:resourcetype/>
   <d:getcontentlength>12</d:getcontentlength>
   <d:getetag>"def456"</d:getetag>
   <d:getlastmodified>Fri, 20 Nov 2026 09:00:00 GMT</d:getlastmodified>
  </d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat>
 </d:response>
</d:multistatus>`,
	"/remote.php/dav/files/app/photos/": `<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:">
 <d:response>
  <d:href>{host}/remote.php/dav/files/app/photos/</d:href>
  <d:propstat><d:prop><d:resourcetype><d:collection/></d:resourcetype></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat>
 </d:response>
 <d:response>
  <d:href>{host}/remote.php/dav/files/app/photos/k%C5%AF%C5%88%20%231.jpg</d:href>
  <d:propstat><d:prop>
   <d:resourcetype/>
   <d:getcontentlength>2048</d:getcontentlength>
   <d:getcontenttype>image/jpeg</d:getcontenttype>
   <d:getetag>W/"abc123"</d:getetag>
   <d:getlastmodified>Fri, 20 Nov 2026 10:00:00 GMT</d:getlastmodified>
  </d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat>
 </d:response>
</d:multistatus>`,
	"/remote.php/dav/files/app/manual.pdf": `<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:">
 <d:response>
  <d:href>/remote.php/dav/files/app/manual.pdf</d:href>
  <d:propstat><d:prop>
   <d:resourcetype/>
   <d:getcontentlength>12</d:getcontentlength>
   <d:getetag>"def456"</d:getetag>
  </d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat>
 </d:response>
</d:multistatus>`,
	"/remote.php/dav/files/app/photos": `<?xml version="1.0"?>
<d:multistatus xmlns:d="DAV:">
 <d:response>
  <d:href>/remote.php/dav/files/app/photos/</d:href>
  <d:propstat><d:prop><d:resourcetype><d:collection/></d:resourcetype></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat>
 </d:response>
</d:multistatus>`,
}

type closeRecorder struct {
	*strings.Reader
	closed bool
}

func (r *closeRecorder) Close() error {
	r.closed = true
	return nil
}

// newServer serves an in-memory WebDAV tree under /dav. allow, when given,
// decides which requests pass; the others get 401.
func newServer(t *testing.T, allow func(r *http.Request) bool) *httptest.Server {
	t.Helper()
	handler := &xwebdav.Handler{
		Prefix:     "/dav",
		FileSystem: xwebdav.NewMemFS(),
		LockSystem: xwebdav.NewMemLS(),
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if allow != nil && !allow(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

func newDriver(t *testing.T, baseURL string, options ...webdav.Option) *webdav.Driver {
	t.Helper()
	driver, err := webdav.NewDriver(baseURL, options...)
	require.NoError(t, err)
	return driver
}

func putText(ctx context.Context, driver *webdav.Driver, key, value string) error {
	return driver.Put(ctx, key, strings.NewReader(value), int64(len(value)), "text/plain")
}

func deleteOnServer(ctx context.Context, t *testing.T, target string) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, target, nil)
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	require.Equal(t, http.StatusNoContent, res.StatusCode)
}

func requireContent(ctx context.Context, t *testing.T, driver *webdav.Driver, key, expected string) {
	t.Helper()
	object, exists, err := driver.Get(ctx, key)
	require.NoError(t, err)
	require.True(t, exists)
	body, err := io.ReadAll(object.Body)
	require.NoError(t, err)
	require.NoError(t, object.Body.Close())
	require.Equal(t, expected, string(body))
}
