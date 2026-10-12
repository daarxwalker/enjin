package session

import (
	"context"
	"fmt"
	"time"

	"github.com/daarxwalker/enjin/security"
)

func getProfileFromCache(ctx context.Context, cache cache, token string, roles map[string]*security.Role) (
	*Profile, error,
) {
	var p cacheProfile
	found, err := cache.GetJSON(ctx, createCacheKey(token), &p)
	if err != nil && found {
		return nil, fmt.Errorf("%w: %w", ErrInvalidProfile, err)
	}
	if err != nil {
		return nil, fmt.Errorf("get json from cache failed: %w", err)
	}
	if !found {
		return nil, nil
	}
	profile, err := cacheToProfile(p, roles)
	if err != nil {
		return nil, fmt.Errorf("%w: map cache to profile failed: %w", ErrInvalidProfile, err)
	}
	return profile, nil
}

func saveProfileToCache(ctx context.Context, cache cache, p *Profile, ttl time.Duration) error {
	if p.ID == "" {
		return nil
	}
	if setErr := cache.SetJSON(ctx, createCacheKey(p.Token), profileToCache(p), ttl); setErr != nil {
		return fmt.Errorf("set json to cache failed: %w", setErr)
	}
	return nil
}
