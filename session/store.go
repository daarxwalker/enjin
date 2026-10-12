package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/daarxwalker/enjin/cookie"
	"github.com/daarxwalker/enjin/security"
	"github.com/daarxwalker/enjin/token"
)

type Store struct {
	cache  cache
	cookie *cookie.Config
	log    *slog.Logger

	roles  map[string]*security.Role
	bearer bool
}

type StoreOption func(*Store)

func NewStore(cache cache, options ...StoreOption) *Store {
	m := &Store{
		cache: cache,
		cookie: cookie.NewConfig(
			cookie.WithName(defaultCookieName),
			cookie.WithTTL(30*24*time.Hour),
		),
		roles: nil,
		log:   slog.Default(),
	}
	for _, opt := range options {
		opt(m)
	}
	return m
}

func (s *Store) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var err error
			ctx := r.Context()
			cv := &ctxVal{
				store: s,
			}
			r = r.WithContext(
				context.WithValue(r.Context(), ctxKey, cv),
			)
			sessionToken, err := s.getToken(r, cv)
			if err != nil {
				s.log.ErrorContext(ctx, fmt.Sprintf("get token from cookies failed: %s", err))
				next.ServeHTTP(w, r)
				return
			}
			if sessionToken == "" {
				next.ServeHTTP(w, r)
				return
			}
			cv.token = sessionToken
			cv.profile, err = getProfileFromCache(ctx, s.cache, sessionToken, s.roles)
			switch {
			case errors.Is(err, ErrInvalidProfile):
				s.log.ErrorContext(ctx, fmt.Sprintf("get session profile failed: %s", err))
				cv.destroyProfile(ctx, w)
			case err != nil:
				s.log.ErrorContext(ctx, fmt.Sprintf("get session profile failed: %s", err))
				cv.profile = nil
			case cv.profile == nil:
				cv.token = ""
				if !cv.bearer {
					s.cookie.Destroy(w)
				}
			default:
				cv.refreshProfile(ctx, w)
			}
			next.ServeHTTP(w, r)
		},
	)
}

func (s *Store) Seed(ctx context.Context, options ...AttachOption) (string, error) {
	p := &Profile{
		Token: token.New(),
	}
	for _, option := range options {
		option(p)
	}
	if err := saveProfileToCache(ctx, s.cache, p, s.cookie.TTL); err != nil {
		return "", err
	}
	return p.Token, nil
}

func (s *Store) getToken(r *http.Request, cv *ctxVal) (string, error) {
	if s.bearer {
		if value := getTokenFromHeader(r); value != "" {
			cv.bearer = true
			return value, nil
		}
	}
	return getTokenFromCookie(r, s.cookie.Name)
}

func (s *Store) Cookie() string {
	return s.cookie.Name
}

func WithRoles(roles map[string]*security.Role) StoreOption {
	return func(s *Store) {
		s.roles = roles
	}
}

func WithLogger(log *slog.Logger) StoreOption {
	return func(s *Store) {
		if log != nil {
			s.log = log
		}
	}
}

func WithBearer() StoreOption {
	return func(s *Store) {
		s.bearer = true
	}
}

func WithConfig(options ...cookie.Option) StoreOption {
	return func(s *Store) {
		s.cookie.Apply(options...)
	}
}
