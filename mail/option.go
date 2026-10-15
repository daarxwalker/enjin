package mail

import (
	"context"
	"crypto/tls"
	"log/slog"
	"time"
)

type SMTPOption func(*SMTPDriver)

// WithSMTPEncryption sets how the connection is encrypted; the default is
// SMTPEncryptionAuto.
func WithSMTPEncryption(encryption SMTPEncryption) SMTPOption {
	return func(d *SMTPDriver) {
		d.encryption = encryption
	}
}

// WithSMTPHelo sets the name sent in EHLO; the default is os.Hostname().
func WithSMTPHelo(name string) SMTPOption {
	return func(d *SMTPDriver) {
		if name != "" {
			d.helo = name
		}
	}
}

// WithSMTPTLSConfig replaces the TLS settings used for STARTTLS and implicit
// TLS (custom CA, another ServerName, InsecureSkipVerify in development). An
// empty ServerName is filled with the host.
func WithSMTPTLSConfig(config *tls.Config) SMTPOption {
	return func(d *SMTPDriver) {
		if config != nil {
			d.tlsConfig = config.Clone()
		}
	}
}

// WithSMTPOAuth2 authenticates with AUTH XOAUTH2 instead of a password. The
// driver calls tokenSource before every send, so it can return a fresh access
// token; the password passed to NewSMTPDriver is ignored.
func WithSMTPOAuth2(tokenSource func(ctx context.Context) (string, error)) SMTPOption {
	return func(d *SMTPDriver) {
		d.tokenSource = tokenSource
	}
}

// WithSMTPTimeout limits one whole send, from dial to QUIT; the default is
// 30 seconds.
func WithSMTPTimeout(timeout time.Duration) SMTPOption {
	return func(d *SMTPDriver) {
		if timeout > 0 {
			d.timeout = timeout
		}
	}
}

// WithSMTPLogger sets the logger for failures that do not fail the send, such
// as a QUIT after the server accepted the message.
func WithSMTPLogger(log *slog.Logger) SMTPOption {
	return func(d *SMTPDriver) {
		if log != nil {
			d.log = log
		}
	}
}
