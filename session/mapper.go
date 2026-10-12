package session

import (
	"fmt"

	"github.com/daarxwalker/enjin/security"
)

func profileToCache(p *Profile) cacheProfile {
	var roleName string
	if p.Role != nil {
		roleName = p.Role.Name
	}
	return cacheProfile{
		ID:     p.ID,
		Key:    p.Key,
		Role:   roleName,
		Token:  p.Token,
		Values: p.Values,
	}
}

func cacheToProfile(c cacheProfile, roles map[string]*security.Role) (*Profile, error) {
	var (
		role    *security.Role
		hasRole bool
	)
	if c.Role != "" {
		role, hasRole = roles[c.Role]
		if !hasRole {
			return nil, fmt.Errorf("role not found: %s", c.Role)
		}
	}
	return &Profile{
		ID:     c.ID,
		Key:    c.Key,
		Role:   role,
		Token:  c.Token,
		Values: c.Values,
	}, nil
}
