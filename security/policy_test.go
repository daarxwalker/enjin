package security

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPolicy(t *testing.T) {
	admin := &Role{Name: "admin", Securables: []Securable{"sync"}}

	t.Run(
		"reject missing role", func(t *testing.T) {
			require.False(t, AnyRole()(nil))
			require.False(t, RequireRole(admin)(nil))
			require.False(t, RequireSecurable("sync")(nil))
			require.False(t, Require(admin, "sync")(nil))
		},
	)

	t.Run(
		"accept missing role", func(t *testing.T) {
			require.True(t, AnyAccount()(nil))
			require.True(t, Not(RequireRole(admin))(nil))
		},
	)

	t.Run(
		"accept matching role", func(t *testing.T) {
			require.True(t, AnyRole()(admin))
			require.True(t, RequireRole(admin)(admin))
			require.True(t, RequireSecurable("sync")(admin))
			require.True(t, Require(admin, "sync")(admin))
		},
	)

	t.Run(
		"reject other role", func(t *testing.T) {
			editor := &Role{Name: "editor"}
			require.False(t, RequireRole(admin)(editor))
			require.False(t, RequireSecurable("sync")(editor))
			require.False(t, Require(admin, "sync")(editor))
		},
	)
}
