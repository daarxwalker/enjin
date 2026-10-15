package mail

import (
	"net/smtp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSMTPDriverCreateAuth(t *testing.T) {
	driver := NewSMTPDriver("smtp.example.com", "587", "user", "pass")

	t.Run(
		"prefers plain over login on a secure connection", func(t *testing.T) {
			mechanisms := map[string]struct{}{smtpAuthLogin: {}, smtpAuthPlain: {}}
			auth, err := driver.createAuth(t.Context(), mechanisms, true)
			require.NoError(t, err)
			mechanism, _, err := auth.Start(&smtp.ServerInfo{Name: "smtp.example.com", TLS: true})
			require.NoError(t, err)
			require.Equal(t, smtpAuthPlain, mechanism)
		},
	)

	t.Run(
		"refuses a cleartext password over an unencrypted connection", func(t *testing.T) {
			mechanisms := map[string]struct{}{smtpAuthLogin: {}, smtpAuthPlain: {}}
			_, err := driver.createAuth(t.Context(), mechanisms, false)
			require.ErrorContains(t, err, "unencrypted connection")
		},
	)

	t.Run(
		"falls back to cram-md5 over an unencrypted connection", func(t *testing.T) {
			mechanisms := map[string]struct{}{smtpAuthPlain: {}, smtpAuthCRAMMD5: {}}
			auth, err := driver.createAuth(t.Context(), mechanisms, false)
			require.NoError(t, err)
			mechanism, _, err := auth.Start(&smtp.ServerInfo{Name: "smtp.example.com"})
			require.NoError(t, err)
			require.Equal(t, smtpAuthCRAMMD5, mechanism)
		},
	)
}
