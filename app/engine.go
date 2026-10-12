package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/daarxwalker/enjin/httpx"
	"github.com/daarxwalker/enjin/security"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
)

type Engine struct {
	Ctx                context.Context
	metrics            http.Handler
	spec               Spec
	cors               *cors.Options
	securityHeaders    *security.Headers
	Root               *Router
	Router             *Router
	cleanups           []func(ctx context.Context) error
	opsToken           string
	host               string
	port               string
	defaultHost        string
	middlewares        []func(http.Handler) http.Handler
	trustedProxies     []netip.Prefix
	allowedHosts       []string
	server             Server
	cloudflare         bool
	useDevTools        bool
	useProfiler        bool
	useCoreMiddlewares bool
}

func NewEngine(options ...Option) (*Engine, error) {
	e := new(Engine{
		useCoreMiddlewares: true,
	})
	for _, option := range options {
		option(e)
	}
	if e.Ctx == nil {
		e.Ctx = context.Background()
	}
	if e.host == "" {
		e.host = "0.0.0.0"
	}
	if e.port == "" {
		e.port = "80"
	}
	if e.Router == nil {
		return nil, errors.New("router was not set")
	}
	if e.spec != nil {
		e.Router.spec = e.spec
	}
	e.server = e.server.withDefaults()
	e.Root = e.Router
	if e.securityHeaders != nil {
		e.Root.Use(e.securityHeaders.Middleware)
	}
	e.Root.Use(limitRequestBody(e.server.MaxBodyBytes))
	if e.useCoreMiddlewares {
		e.registerCoreMiddlewares()
	}
	if e.useDevTools {
		e.registerDevTools()
	}
	if len(e.middlewares) > 0 {
		e.Router = e.Root.With(e.middlewares...)
	}
	return e, nil
}

func (e *Engine) Start() error {
	if e.metrics != nil || e.useProfiler {
		e.registerOps()
	}
	if err := e.createSpecRoute(); err != nil {
		return fmt.Errorf("create spec route failed: %w", err)
	}
	if err := e.Router.Err(); err != nil {
		return fmt.Errorf("register routes failed: %w", err)
	}
	server := e.createServer()
	ctx, stop := signal.NotifyContext(e.Ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	served := make(chan error, 1)
	go func() {
		served <- e.serve(server)
	}()
	select {
	case err := <-served:
		return errors.Join(err, e.runCleanup())
	case <-ctx.Done():
	}
	return e.shutdown(server, served)
}

func (e *Engine) registerCoreMiddlewares() {
	e.Root.Use(
		middleware.Recoverer,
		middleware.Heartbeat("/healthz"),
		e.createProxy().Middleware,
	)
	if e.cors != nil {
		e.Root.Use(cors.Handler(*e.cors))
	}
	e.Root.Use(middleware.Compress(5))
}

func (e *Engine) createProxy() *httpx.Proxy {
	options := []httpx.ProxyOption{
		httpx.WithTrustedProxies(e.trustedProxies...),
		httpx.WithCloudflare(e.cloudflare),
	}
	if len(e.allowedHosts) > 0 {
		options = append(options, httpx.WithAllowedHosts(e.allowedHosts...))
	}
	if e.defaultHost != "" {
		options = append(options, httpx.WithDefaultHost(e.defaultHost))
	}
	if e.opsToken != "" && e.metrics != nil {
		options = append(options, httpx.WithHostExemptPaths("/metrics"))
	}
	if e.opsToken != "" && e.useProfiler {
		options = append(options, httpx.WithHostExemptPaths("/debug"))
	}
	return httpx.NewProxy(options...)
}

func (e *Engine) registerOps() {
	if e.opsToken == "" {
		log.Printf("🔴 ops token not set, metrics and profiler disabled\n")
		return
	}
	e.Root.Raw().Group(
		func(r chi.Router) {
			r.Use(
				httprate.LimitBy(100, time.Minute, func(r *http.Request) (string, error) {
					return httpx.ClientIP(r), nil
				}),
				security.RequireBearer(e.opsToken),
			)
			if e.metrics != nil {
				r.Handle("/metrics", e.metrics)
			}
			if e.useProfiler {
				r.Mount("/debug", middleware.Profiler())
			}
		},
	)
}

func (e *Engine) registerDevTools() {
	e.Root.Use(middleware.Logger)
}

func (e *Engine) createServer() *http.Server {
	server := &http.Server{
		ReadHeaderTimeout: e.server.ReadHeaderTimeout,
		ReadTimeout:       e.server.ReadTimeout,
		WriteTimeout:      e.server.WriteTimeout,
		IdleTimeout:       e.server.IdleTimeout,
		MaxHeaderBytes:    e.server.MaxHeaderBytes,
		Addr:              e.host + ":" + e.port,
		Handler:           e.Root.Raw(),
	}
	return server
}

func (e *Engine) serve(server *http.Server) error {
	log.Printf("🌐 started on host: %s\n", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen on %s failed: %w", server.Addr, err)
	}
	return nil
}

func (e *Engine) shutdown(server *http.Server, served <-chan error) error {
	log.Println("⏳ stopping...")
	ctx, cancel := context.WithTimeout(context.WithoutCancel(e.Ctx), 5*time.Second)
	defer cancel()
	var shutdownErr error
	if err := server.Shutdown(ctx); err != nil {
		shutdownErr = fmt.Errorf("shutdown server failed: %w", err)
	}
	err := errors.Join(shutdownErr, <-served, e.runCleanup())
	if err == nil {
		log.Println("✅ stopped")
	}
	return err
}

func (e *Engine) runCleanup() error {
	if len(e.cleanups) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(e.Ctx), 5*time.Second)
	defer cancel()
	var errs []error
	for _, cleanup := range e.cleanups {
		if err := cleanup(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}
	return nil
}

func (e *Engine) createSpecRoute() error {
	if e.Router.spec == nil {
		return nil
	}
	marshaled, err := e.Router.spec.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal spec failed: %w", err)
	}
	e.Router.Get("spec", "/spec.json", func(w http.ResponseWriter, _ *http.Request) error {
		return httpx.JSONBytes(w, http.StatusOK, marshaled)
	})
	return nil
}

type Option func(*Engine)

func WithContext(ctx context.Context) Option {
	return func(e *Engine) {
		e.Ctx = ctx
	}
}

func WithHost(host string) Option {
	return func(e *Engine) {
		e.host = host
	}
}

func WithPort(port uint64) Option {
	return func(e *Engine) {
		e.port = strconv.FormatUint(port, 10)
	}
}

func WithRouter(router *Router) Option {
	return func(e *Engine) {
		e.Router = router
	}
}

func WithSpec(spec Spec) Option {
	return func(e *Engine) {
		e.spec = spec
	}
}

func WithCORS(options cors.Options) Option {
	return func(e *Engine) {
		e.cors = &options
	}
}

func WithServer(server Server) Option {
	return func(e *Engine) {
		e.server = server
	}
}

func WithSecurityHeaders(headers *security.Headers) Option {
	return func(e *Engine) {
		e.securityHeaders = headers
	}
}

func WithCoreMiddlewares(use bool) Option {
	return func(e *Engine) {
		e.useCoreMiddlewares = use
	}
}

func WithDevTools(use bool) Option {
	return func(e *Engine) {
		e.useDevTools = use
	}
}

func WithMetrics(handler http.Handler) Option {
	return func(e *Engine) {
		e.metrics = handler
	}
}

func WithProfiler(use bool) Option {
	return func(e *Engine) {
		e.useProfiler = use
	}
}

func WithOpsToken(token string) Option {
	return func(e *Engine) {
		e.opsToken = token
	}
}

func WithMiddleware(middlewares ...func(http.Handler) http.Handler) Option {
	return func(e *Engine) {
		e.middlewares = append(e.middlewares, middlewares...)
	}
}

func WithTrustedProxies(prefixes ...netip.Prefix) Option {
	return func(e *Engine) {
		e.trustedProxies = append(e.trustedProxies, prefixes...)
	}
}

func WithCloudflare(enabled bool) Option {
	return func(e *Engine) {
		e.cloudflare = enabled
	}
}

func WithAllowedHosts(hosts ...string) Option {
	return func(e *Engine) {
		e.allowedHosts = append(e.allowedHosts, hosts...)
	}
}

func WithDefaultHost(host string) Option {
	return func(e *Engine) {
		e.defaultHost = host
	}
}

func WithCleanup(cleanup func(ctx context.Context) error) Option {
	return func(e *Engine) {
		e.cleanups = append(e.cleanups, cleanup)
	}
}
