package session

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type ctxKeyType struct{}

var ctxKey = ctxKeyType{}

type ctxVal struct {
	store   *Store
	profile *Profile
	token   string
	bearer  bool
}

func (c *ctxVal) saveProfile(ctx context.Context, w http.ResponseWriter) {
	if c.profile == nil || c.profile.ID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := saveProfileToCache(ctx, c.store.cache, c.profile, c.store.cookie.TTL); err != nil {
		c.store.log.ErrorContext(ctx, fmt.Sprintf("save profile to cache failed: %s", err))
		return
	}
	if !c.bearer {
		c.store.cookie.Attach(w, c.profile.Token)
	}
}

func (c *ctxVal) refreshProfile(ctx context.Context, w http.ResponseWriter) {
	if c.profile == nil || c.profile.ID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	updated, err := c.store.cache.UpdateExpiration(ctx, createCacheKey(c.token), c.store.cookie.TTL)
	if err != nil {
		c.store.log.ErrorContext(ctx, fmt.Sprintf("update session expiration in cache failed: %s", err))
		return
	}
	if !updated {
		c.profile = nil
		c.token = ""
		if !c.bearer {
			c.store.cookie.Destroy(w)
		}
		return
	}
	if !c.bearer {
		c.store.cookie.Attach(w, c.token)
	}
}

func (c *ctxVal) replaceProfile(ctx context.Context, w http.ResponseWriter, next *Profile) {
	if c.token != "" && c.token != next.Token {
		deleteCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := c.store.cache.Delete(deleteCtx, createCacheKey(c.token)); err != nil {
			c.store.log.ErrorContext(ctx, fmt.Sprintf("delete replaced session in cache failed: %s", err))
		}
	}
	c.profile = next
	c.token = next.Token
	c.saveProfile(ctx, w)
}

func (c *ctxVal) destroyProfile(ctx context.Context, w http.ResponseWriter) {
	token := c.token
	c.profile = nil
	c.token = ""
	if token == "" {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := c.store.cache.Delete(ctx, createCacheKey(token)); err != nil {
		c.store.log.ErrorContext(ctx, fmt.Sprintf("delete session in cache failed: %s", err))
	}
	if !c.bearer {
		c.store.cookie.Destroy(w)
	}
}
