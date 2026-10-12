package app

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/security"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func TestEngine(t *testing.T) {
	t.Run("without router", func(t *testing.T) {
		if _, err := NewEngine(); err == nil {
			t.Fatal("NewEngine without router succeeded, want error")
		}
	})

	t.Run("duplicate route", func(t *testing.T) {
		cleaned := false
		e := newTestEngine(t,
			WithRouter(NewRouter(chi.NewRouter(), NewDefaultHandler())),
			WithCleanup(func(context.Context) error {
				cleaned = true
				return nil
			}),
		)
		e.Router.Get("home", "/", noContent)
		e.Router.Get("home", "/other", noContent)
		if err := e.Start(); err == nil {
			t.Fatal("Start with duplicate route succeeded, want error")
		}
		if cleaned {
			t.Fatal("cleanup ran although server never started")
		}
	})

	t.Run("listen failed", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = listener.Close()
		}()
		cleaned := false
		e := newTestEngine(t,
			WithRouter(NewRouter(chi.NewRouter(), NewDefaultHandler())),
			WithHost("127.0.0.1"),
			WithPort(listenerPort(t, listener)),
			WithCleanup(func(context.Context) error {
				cleaned = true
				return nil
			}),
		)
		if err = e.Start(); err == nil {
			t.Fatal("Start on occupied port succeeded, want error")
		}
		if !cleaned {
			t.Fatal("cleanup did not run")
		}
	})

	t.Run("runs every cleanup in order", func(t *testing.T) {
		firstErr := errors.New("first")
		var order []string
		e := newTestEngine(t,
			WithRouter(NewRouter(chi.NewRouter(), NewDefaultHandler())),
			WithCleanup(func(context.Context) error {
				order = append(order, "first")
				return firstErr
			}),
			WithCleanup(func(context.Context) error {
				order = append(order, "second")
				return nil
			}),
		)
		if err := e.runCleanup(); !errors.Is(err, firstErr) {
			t.Fatalf("runCleanup = %v, want first error", err)
		}
		if len(order) != 2 || order[0] != "first" || order[1] != "second" {
			t.Fatalf("order = %v, want [first second]", order)
		}
	})

	t.Run("stops on context cancel", func(t *testing.T) {
		port := freePort(t)
		addr := "127.0.0.1:" + strconv.FormatUint(port, 10)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		cleanupErr := errors.New("cleanup")
		listening := true
		e := newTestEngine(t,
			WithContext(ctx),
			WithRouter(NewRouter(chi.NewRouter(), NewDefaultHandler())),
			WithHost("127.0.0.1"),
			WithPort(port),
			WithCleanup(func(context.Context) error {
				conn, dialErr := net.Dial("tcp", addr)
				if dialErr == nil {
					_ = conn.Close()
				}
				listening = dialErr == nil
				return cleanupErr
			}),
		)
		started := make(chan error, 1)
		go func() {
			started <- e.Start()
		}()
		waitForHealthz(t, addr)
		cancel()
		select {
		case err := <-started:
			if !errors.Is(err, cleanupErr) {
				t.Fatalf("Start = %v, want cleanup error", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("Start did not return after context cancel")
		}
		if listening {
			t.Fatal("cleanup ran before server shutdown")
		}
	})

	t.Run("allowed hosts", func(t *testing.T) {
		e := newTestEngine(t,
			WithRouter(NewRouter(chi.NewRouter(), NewDefaultHandler())),
			WithAllowedHosts("example.com"),
			WithMetrics(http.HandlerFunc(serveMetrics)),
			WithOpsToken("secret"),
		)
		e.Router.Get("home", "/", noContent)
		e.registerOps()

		tests := []struct {
			name string
			host string
			path string
			want int
		}{
			{"allowed host", "example.com", "/", http.StatusNoContent},
			{"other host", "evil.example", "/", http.StatusMisdirectedRequest},
			{"healthz on other host", "app.example:80", "/healthz", http.StatusOK},
			{"metrics on other host", "app.example:80", "/metrics", http.StatusOK},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := httptest.NewRequest(http.MethodGet, tt.path, nil)
				r.Host = tt.host
				r.Header.Set("Authorization", "Bearer secret")
				w := httptest.NewRecorder()
				e.Root.Raw().ServeHTTP(w, r)
				if w.Code != tt.want {
					t.Fatalf("GET %s on %s = %d, want %d", tt.path, tt.host, w.Code, tt.want)
				}
			})
		}
	})

	t.Run("cors", func(t *testing.T) {
		tests := []struct {
			name    string
			options []Option
			origin  string
			want    string
		}{
			{"allowed origin", []Option{WithCORS(cors.Options{AllowedOrigins: []string{"https://app.example"}})}, "https://app.example", "https://app.example"},
			{"other origin", []Option{WithCORS(cors.Options{AllowedOrigins: []string{"https://app.example"}})}, "https://evil.example", ""},
			{"without cors", nil, "https://app.example", ""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := newTestEngine(t, append([]Option{WithRouter(NewRouter(chi.NewRouter(), NewDefaultHandler()))}, tt.options...)...)
				e.Router.Post("item", "/items", noContent)
				r := httptest.NewRequest(http.MethodOptions, "/items", nil)
				r.Header.Set("Origin", tt.origin)
				r.Header.Set("Access-Control-Request-Method", http.MethodPost)
				w := httptest.NewRecorder()
				e.Root.Raw().ServeHTTP(w, r)
				if got := w.Header().Get("Access-Control-Allow-Origin"); got != tt.want {
					t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, tt.want)
				}
			})
		}
	})

	t.Run("server", func(t *testing.T) {
		tests := []struct {
			name    string
			options []Option
			want    Server
		}{
			{"defaults", nil, defaultServer},
			{
				"partial override",
				[]Option{WithServer(Server{WriteTimeout: 5 * time.Minute, MaxBodyBytes: 1 << 10})},
				Server{
					ReadHeaderTimeout: defaultServer.ReadHeaderTimeout,
					ReadTimeout:       defaultServer.ReadTimeout,
					WriteTimeout:      5 * time.Minute,
					IdleTimeout:       defaultServer.IdleTimeout,
					MaxHeaderBytes:    defaultServer.MaxHeaderBytes,
					MaxBodyBytes:      1 << 10,
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := newTestEngine(t, append([]Option{WithRouter(NewRouter(chi.NewRouter(), NewDefaultHandler()))}, tt.options...)...)
				server := e.createServer()
				got := Server{
					ReadHeaderTimeout: server.ReadHeaderTimeout,
					ReadTimeout:       server.ReadTimeout,
					WriteTimeout:      server.WriteTimeout,
					IdleTimeout:       server.IdleTimeout,
					MaxHeaderBytes:    server.MaxHeaderBytes,
					MaxBodyBytes:      e.server.MaxBodyBytes,
				}
				if got != tt.want {
					t.Fatalf("server = %+v, want %+v", got, tt.want)
				}
			})
		}
	})

	t.Run("body limit", func(t *testing.T) {
		tests := []struct {
			name          string
			body          string
			contentLength int64
			want          int
		}{
			{"within limit", strings.Repeat("a", 8), 8, http.StatusNoContent},
			{"declared over limit", strings.Repeat("a", 32), 32, http.StatusRequestEntityTooLarge},
			{"streamed over limit", strings.Repeat("a", 32), -1, http.StatusRequestEntityTooLarge},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := newTestEngine(t, WithRouter(NewRouter(chi.NewRouter(), NewDefaultHandler())), WithServer(Server{MaxBodyBytes: 16}))
				e.Router.Post("item", "/items", func(w http.ResponseWriter, r *http.Request) error {
					if _, err := io.ReadAll(r.Body); err != nil {
						return &httpx.Err{Status: http.StatusRequestEntityTooLarge}
					}
					w.WriteHeader(http.StatusNoContent)
					return nil
				})
				r := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(tt.body))
				r.ContentLength = tt.contentLength
				w := httptest.NewRecorder()
				e.Root.Raw().ServeHTTP(w, r)
				if w.Code != tt.want {
					t.Fatalf("status = %d, want %d", w.Code, tt.want)
				}
			})
		}
	})

	t.Run("security headers", func(t *testing.T) {
		tests := []struct {
			name    string
			options []Option
			path    string
			want    string
		}{
			{"feature route", []Option{WithSecurityHeaders(security.NewHeaders())}, "/items", "nosniff"},
			{"root route", []Option{WithSecurityHeaders(security.NewHeaders())}, "/static", "nosniff"},
			{"healthz", []Option{WithSecurityHeaders(security.NewHeaders())}, "/healthz", "nosniff"},
			{"without headers", nil, "/items", ""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := newTestEngine(t, append([]Option{WithRouter(NewRouter(chi.NewRouter(), NewDefaultHandler()))}, tt.options...)...)
				e.Router.Get("item", "/items", noContent)
				e.Root.Raw().Get("/static", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
				w := httptest.NewRecorder()
				e.Root.Raw().ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))
				if got := w.Header().Get("X-Content-Type-Options"); got != tt.want {
					t.Fatalf("X-Content-Type-Options = %q, want %q", got, tt.want)
				}
			})
		}
	})

	t.Run("ops routes", func(t *testing.T) {
		tests := []struct {
			name          string
			options       []Option
			path          string
			authorization string
			want          int
		}{
			{"metrics without header", []Option{WithMetrics(http.HandlerFunc(serveMetrics)), WithOpsToken("secret")}, "/metrics", "", http.StatusUnauthorized},
			{"metrics with wrong token", []Option{WithMetrics(http.HandlerFunc(serveMetrics)), WithOpsToken("secret")}, "/metrics", "Bearer other", http.StatusUnauthorized},
			{"metrics with token", []Option{WithMetrics(http.HandlerFunc(serveMetrics)), WithOpsToken("secret")}, "/metrics", "Bearer secret", http.StatusOK},
			{"metrics without ops token", []Option{WithMetrics(http.HandlerFunc(serveMetrics))}, "/metrics", "Bearer ", http.StatusNotFound},
			{"profiler with token", []Option{WithProfiler(true), WithOpsToken("secret")}, "/debug/pprof/", "Bearer secret", http.StatusOK},
			{"profiler without header", []Option{WithProfiler(true), WithOpsToken("secret")}, "/debug/pprof/", "", http.StatusUnauthorized},
			{"profiler not on legacy path", []Option{WithProfiler(true), WithOpsToken("secret")}, "/profiler/pprof/", "Bearer secret", http.StatusNotFound},
			{"profiler off by default", []Option{WithMetrics(http.HandlerFunc(serveMetrics)), WithOpsToken("secret")}, "/debug/pprof/", "Bearer secret", http.StatusNotFound},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				e := newTestEngine(t, append([]Option{WithRouter(NewRouter(chi.NewRouter(), NewDefaultHandler()))}, tt.options...)...)
				e.registerOps()
				r := httptest.NewRequest(http.MethodGet, tt.path, nil)
				if tt.authorization != "" {
					r.Header.Set("Authorization", tt.authorization)
				}
				w := httptest.NewRecorder()
				e.Root.Raw().ServeHTTP(w, r)
				if w.Code != tt.want {
					t.Fatalf("GET %s = %d, want %d", tt.path, w.Code, tt.want)
				}
			})
		}
	})

	t.Run("use after route registration is an error", func(t *testing.T) {
		e := newTestEngine(t, WithRouter(NewRouter(chi.NewRouter(), NewDefaultHandler())))
		e.Router.Get("home", "/", noContent)
		e.Router.Use(markAppMiddleware)
		if err := e.Start(); err == nil || !strings.Contains(err.Error(), "use middlewares failed") {
			t.Fatalf("Start() = %v, want use middlewares error", err)
		}
	})

	t.Run("use after new engine with ops routes", func(t *testing.T) {
		e := newTestEngine(t,
			WithRouter(NewRouter(chi.NewRouter(), NewDefaultHandler())),
			WithMetrics(http.HandlerFunc(serveMetrics)),
			WithOpsToken("secret"),
		)
		e.Root.Use(markAppMiddleware)
		e.Router.Get("home", "/", noContent)
		if err := e.Router.Err(); err != nil {
			t.Fatalf("Err() = %v, want nil", err)
		}
		e.registerOps()
		r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		r.Header.Set("Authorization", "Bearer secret")
		w := httptest.NewRecorder()
		e.Root.Raw().ServeHTTP(w, r)
		if w.Code != http.StatusOK || w.Header().Get(appMarkerHeader) != "1" {
			t.Fatalf("GET /metrics = %d with marker %q, want 200 with marker", w.Code, w.Header().Get(appMarkerHeader))
		}
	})

	t.Run("dev mode without profiler", func(t *testing.T) {
		e := newTestEngine(t,
			WithRouter(NewRouter(chi.NewRouter(), NewDefaultHandler())),
			WithDevTools(true),
			WithMetrics(http.HandlerFunc(serveMetrics)),
			WithOpsToken("secret"),
		)
		e.registerOps()
		for _, path := range []string{"/debug/pprof/", "/profiler/pprof/"} {
			r := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			e.Root.Raw().ServeHTTP(w, r)
			if w.Code != http.StatusNotFound {
				t.Fatalf("GET %s = %d, want %d", path, w.Code, http.StatusNotFound)
			}
		}
	})

	t.Run("root routes skip app middlewares", func(t *testing.T) {
		e := newTestEngine(t,
			WithRouter(NewRouter(chi.NewRouter(), NewDefaultHandler())),
			WithMiddleware(markAppMiddleware),
		)
		e.Router.Get("home", "/", noContent)
		e.Router.NotFound(notFound)
		if err := ServeImmutableFS(e.Root, http.FS(fstest.MapFS{"app.abc123.js": {Data: []byte("x")}}), "/public", ""); err != nil {
			t.Fatal(err)
		}
		tests := []struct {
			name             string
			path             string
			wantCode         int
			wantAppMarker    string
			wantCacheControl string
		}{
			{"app route runs app middlewares", "/", http.StatusNoContent, "1", ""},
			{"static file skips app middlewares", "/public/app.abc123.js", http.StatusOK, "", "public, max-age=31536000, immutable"},
			{"not found runs app middlewares", "/missing", http.StatusNotFound, "1", ""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := httptest.NewRequest(http.MethodGet, tt.path, nil)
				w := httptest.NewRecorder()
				e.Root.Raw().ServeHTTP(w, r)
				if w.Code != tt.wantCode {
					t.Fatalf("GET %s = %d, want %d", tt.path, w.Code, tt.wantCode)
				}
				if got := w.Header().Get(appMarkerHeader); got != tt.wantAppMarker {
					t.Errorf("%s = %q, want %q", appMarkerHeader, got, tt.wantAppMarker)
				}
				if got := w.Header().Get("Cache-Control"); got != tt.wantCacheControl {
					t.Errorf("Cache-Control = %q, want %q", got, tt.wantCacheControl)
				}
			})
		}
	})
}

func newTestEngine(t *testing.T, options ...Option) *Engine {
	t.Helper()
	e, err := NewEngine(options...)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

const appMarkerHeader = "X-App-Middleware"

func markAppMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(appMarkerHeader, "1")
		next.ServeHTTP(w, r)
	})
}

func notFound(w http.ResponseWriter, _ *http.Request) error {
	w.WriteHeader(http.StatusNotFound)
	return nil
}

func noContent(w http.ResponseWriter, _ *http.Request) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func freePort(t *testing.T) uint64 {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = listener.Close()
	}()
	return listenerPort(t, listener)
}

func listenerPort(t *testing.T, listener net.Listener) uint64 {
	t.Helper()
	_, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.ParseUint(portText, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	return port
}

func waitForHealthz(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr + "/healthz")
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("server on %s did not start", addr)
}

func serveMetrics(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
