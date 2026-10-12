package session

import (
	"testing"

	"github.com/daarxwalker/enjin/security"

	"github.com/stretchr/testify/require"
)

func TestMapper(t *testing.T) {
	t.Run(
		"round trip", func(t *testing.T) {
			admin := &security.Role{Name: "admin"}
			profile := &Profile{
				ID:     "user",
				Key:    "key",
				Role:   admin,
				Token:  "token",
				Values: map[string]string{"locale": "cs"},
			}
			got, err := cacheToProfile(profileToCache(profile), map[string]*security.Role{admin.Name: admin})
			require.NoError(t, err)
			require.Equal(t, profile, got)
		},
	)
}
