package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/daarxwalker/enjin/app/routespec"

	"github.com/go-chi/chi/v5"
)

type Router struct {
	raw      chi.Router
	handler  Handler
	spec     Spec
	routeSet map[string]*Route
	errs     *[]error
	pattern  string
	langs    []string
}

type Route struct {
	paramSet map[string]struct{}
	method   string
	name     string
	pattern  string
}

func NewRouter(raw chi.Router, handler Handler, options ...RouterOption) *Router {
	r := &Router{
		raw:      raw,
		handler:  handler,
		routeSet: make(map[string]*Route),
		errs:     new([]error),
	}
	for _, option := range options {
		option(r)
	}
	r.Use(r.middleware)
	return r
}

func (r *Router) Raw() chi.Router {
	return r.raw
}

func (r *Router) Err() error {
	return errors.Join(*r.errs...)
}

func (r *Router) With(middlewares ...func(http.Handler) http.Handler) *Router {
	return &Router{
		raw:      r.raw.With(middlewares...),
		handler:  r.handler,
		spec:     r.spec,
		langs:    r.langs,
		pattern:  r.pattern,
		routeSet: r.routeSet,
		errs:     r.errs,
	}
}

func (r *Router) Use(middlewares ...func(http.Handler) http.Handler) {
	defer func() {
		if p := recover(); p != nil {
			r.recordError(fmt.Errorf("use middlewares failed: %v", p))
		}
	}()
	r.raw.Use(middlewares...)
}

func (r *Router) Mount(pattern string, raw http.Handler) {
	r.raw.Mount(pattern, raw)
}

func (r *Router) Group(pattern string, fn func(r *Router)) *Router {
	group := &Router{
		handler:  r.handler,
		spec:     r.spec,
		langs:    r.langs,
		pattern:  r.joinPatterns(r.pattern, pattern),
		routeSet: r.routeSet,
		errs:     r.errs,
	}
	group.raw = r.raw.Route(pattern, func(chr chi.Router) {
		group.raw = chr
		fn(group)
	})
	return group
}

func (r *Router) Get(name, pattern string, handler func(w http.ResponseWriter, r *http.Request) error, specs ...routespec.Option) {
	routePattern := r.joinPatterns(r.pattern, pattern)
	route := r.registerRoute(http.MethodGet, name, routePattern)
	r.registerOperation(http.MethodGet, routePattern, specs...)
	r.raw.With(attachRoute(route)).Get(pattern, r.handler.Handle(handler))
}

func (r *Router) Post(name, pattern string, handler func(w http.ResponseWriter, r *http.Request) error, specs ...routespec.Option) {
	routePattern := r.joinPatterns(r.pattern, pattern)
	route := r.registerRoute(http.MethodPost, name, routePattern)
	r.registerOperation(http.MethodPost, routePattern, specs...)
	r.raw.With(attachRoute(route)).Post(pattern, r.handler.Handle(handler))
}

func (r *Router) Put(name, pattern string, handler func(w http.ResponseWriter, r *http.Request) error, specs ...routespec.Option) {
	routePattern := r.joinPatterns(r.pattern, pattern)
	route := r.registerRoute(http.MethodPut, name, routePattern)
	r.registerOperation(http.MethodPut, routePattern, specs...)
	r.raw.With(attachRoute(route)).Put(pattern, r.handler.Handle(handler))
}

func (r *Router) Patch(name, pattern string, handler func(w http.ResponseWriter, r *http.Request) error, specs ...routespec.Option) {
	routePattern := r.joinPatterns(r.pattern, pattern)
	route := r.registerRoute(http.MethodPatch, name, routePattern)
	r.registerOperation(http.MethodPatch, routePattern, specs...)
	r.raw.With(attachRoute(route)).Patch(pattern, r.handler.Handle(handler))
}

func (r *Router) Delete(name, pattern string, handler func(w http.ResponseWriter, r *http.Request) error, specs ...routespec.Option) {
	routePattern := r.joinPatterns(r.pattern, pattern)
	route := r.registerRoute(http.MethodDelete, name, routePattern)
	r.registerOperation(http.MethodDelete, routePattern, specs...)
	r.raw.With(attachRoute(route)).Delete(pattern, r.handler.Handle(handler))
}

func (r *Router) NotFound(handler func(w http.ResponseWriter, r *http.Request) error) {
	r.raw.NotFound(r.handler.Handle(handler))
}

func (r *Router) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, req *http.Request) {
			ctxVal := &RouterContextValue{
				Router: r,
			}
			ctx := req.Context()
			ctx = context.WithValue(ctx, routerCtxKey, ctxVal)
			next.ServeHTTP(w, req.WithContext(ctx))
		},
	)
}

func (r *Router) registerRoute(method, name, pattern string) *Route {
	if current, ok := r.routeSet[name]; ok && current.method == method {
		r.recordError(errors.New("route '" + name + "' already registered"))
	}
	route := &Route{
		method:   method,
		name:     name,
		pattern:  pattern,
		paramSet: make(map[string]struct{}, strings.Count(pattern, "{")),
	}
	for param := range strings.SplitSeq(pattern, "/") {
		if param == "" || !strings.HasPrefix(param, "{") || !strings.HasSuffix(param, "}") {
			continue
		}
		param = strings.Trim(param, "{}")
		route.paramSet[param] = struct{}{}
	}
	r.routeSet[name] = route
	return route
}

func (r *Router) registerOperation(method, pattern string, specs ...routespec.Option) {
	if r.spec == nil {
		return
	}
	var operation routespec.Operation
	for _, s := range specs {
		s(&operation)
	}
	if r.langs != nil {
		routespec.WithAcceptLanguage(r.langs...)(&operation)
	}
	if err := r.spec.AddOperation(method, pattern, operation); err != nil {
		r.recordError(err)
	}
}

func (r *Router) joinPatterns(patterns ...string) string {
	for i, pattern := range patterns {
		patterns[i] = strings.Trim(pattern, "/")
	}
	result := strings.Join(patterns, "/")
	if !strings.HasPrefix(result, "/") {
		result = "/" + result
	}
	return result
}

func (r *Router) recordError(err error) {
	*r.errs = append(*r.errs, fmt.Errorf("register endpoint failed: %w", err))
}

type RouterOption func(*Router)

func WithRouterLangs(langs ...string) RouterOption {
	return func(r *Router) {
		r.langs = langs
	}
}

func WithRouterSpec(spec Spec) RouterOption {
	return func(r *Router) {
		r.spec = spec
	}
}
