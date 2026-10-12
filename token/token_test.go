package token

import (
	"crypto/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToken(t *testing.T) {
	t.Run(
		"create valid token", func(t *testing.T) {
			first, second := New(), New()
			require.Len(t, first, Length)
			require.True(t, Valid(first))
			require.NotEqual(t, first, second)
		},
	)

	t.Run(
		"reject invalid token", func(t *testing.T) {
			valid := New()
			for _, invalid := range []string{
				"",
				valid[:Length-1],
				valid + "A",
				"+" + valid[1:],
				"/" + valid[1:],
				valid[:Length-1] + "=",
				" " + valid[1:],
				rand.Text(),
				"ABCDEFGHabcdefgh",
				strings.Repeat("A", 4000),
			} {
				require.False(t, Valid(invalid), invalid)
			}
		},
	)
}
