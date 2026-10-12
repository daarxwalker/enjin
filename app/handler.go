package app

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/daarxwalker/enjin/httpx"
)

type DefaultHandler struct {
	log     *slog.Logger
	onError func(w http.ResponseWriter, r *http.Request, err *httpx.Err)
}

func NewDefaultHandler(options ...HandlerOption) *DefaultHandler {
	h := &DefaultHandler{log: slog.Default()}
	for _, option := range options {
		option(h)
	}
	return h
}

func (h *DefaultHandler) Handle(fn func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				var err error
				switch v := rec.(type) {
				case error:
					err = v
					if errors.Is(err, http.ErrAbortHandler) {
						panic(rec)
					}
				default:
					err = fmt.Errorf("%v", v)
				}
				h.logError(r, http.StatusInternalServerError, err, slog.String("stack", string(debug.Stack())))
				h.respond(w, r, &httpx.Err{Status: http.StatusInternalServerError})
			}
		}()
		if err := fn(w, r); err != nil {
			h.handleError(w, r, err)
		}
	}
}

func (h *DefaultHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	appErr, ok := errors.AsType[*httpx.Err](err)
	if !ok {
		appErr = &httpx.Err{Status: http.StatusInternalServerError}
	}
	if appErr.Status >= http.StatusInternalServerError {
		h.logError(r, appErr.Status, err)
	}
	h.respond(w, r, appErr)
}

func (h *DefaultHandler) respond(w http.ResponseWriter, r *http.Request, appErr *httpx.Err) {
	if h.onError != nil {
		h.onError(w, r, appErr)
		return
	}
	if appErr.Status >= http.StatusInternalServerError {
		http.Error(w, http.StatusText(appErr.Status), appErr.Status)
		return
	}
	http.Error(w, appErr.Error(), appErr.Status)
}

func (h *DefaultHandler) logError(r *http.Request, status int, err error, attrs ...any) {
	h.log.ErrorContext(r.Context(), err.Error(), append([]any{
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Int("status", status),
	}, attrs...)...)
}

type HandlerOption func(*DefaultHandler)

func WithHandlerOnError(onError func(w http.ResponseWriter, r *http.Request, err *httpx.Err)) HandlerOption {
	return func(h *DefaultHandler) {
		h.onError = onError
	}
}

func WithHandlerLogger(logger *slog.Logger) HandlerOption {
	return func(h *DefaultHandler) {
		if logger != nil {
			h.log = logger
		}
	}
}
