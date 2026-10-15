package mail

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// SMTPEncryption says how the driver encrypts the connection to the server.
type SMTPEncryption int

const (
	// SMTPEncryptionAuto uses implicit TLS on port 465; on any other port it
	// connects unencrypted and switches to TLS when the server offers
	// STARTTLS. The default.
	SMTPEncryptionAuto SMTPEncryption = iota
	// SMTPEncryptionNone never encrypts, even when the server offers
	// STARTTLS. Meant for local relays such as Mailpit or MailHog.
	SMTPEncryptionNone
	// SMTPEncryptionStartTLS requires STARTTLS and fails when the server does
	// not offer it.
	SMTPEncryptionStartTLS
	// SMTPEncryptionImplicitTLS speaks TLS from the first byte (SMTPS, port
	// 465).
	SMTPEncryptionImplicitTLS
)

type SMTPDriver struct {
	log         *slog.Logger
	tlsConfig   *tls.Config
	tokenSource func(ctx context.Context) (string, error)
	addr        string
	host        string
	port        string
	username    string
	password    string
	helo        string
	encryption  SMTPEncryption
	timeout     time.Duration
}

const (
	defaultSMTPTimeout = 30 * time.Second
	defaultSMTPHelo    = "localhost"
	implicitTLSPort    = "465"
)

// Authentication mechanisms in the order the driver prefers them.
const (
	smtpAuthPlain   = "PLAIN"
	smtpAuthLogin   = "LOGIN"
	smtpAuthCRAMMD5 = "CRAM-MD5"
	smtpAuthXOAuth2 = "XOAUTH2"
)

// NewSMTPDriver sends mail through an SMTP server. An empty username skips
// authentication (relays allowed by IP, Mailpit, MailHog); otherwise the
// driver picks PLAIN, LOGIN or CRAM-MD5, whichever the server offers first in
// that order, and never sends a cleartext password over an unencrypted
// connection to a remote host.
func NewSMTPDriver(
	host string,
	port string,
	username string,
	password string,
	options ...SMTPOption,
) *SMTPDriver {
	d := &SMTPDriver{
		log:       slog.Default(),
		tlsConfig: &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12},
		host:      host,
		port:      port,
		addr:      net.JoinHostPort(host, port),
		username:  username,
		password:  password,
		helo:      defaultSMTPHelo,
		timeout:   defaultSMTPTimeout,
	}
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		d.helo = hostname
	}
	for _, option := range options {
		option(d)
	}
	if d.tlsConfig.ServerName == "" {
		d.tlsConfig.ServerName = host
	}
	if d.encryption == SMTPEncryptionAuto && port == implicitTLSPort {
		d.encryption = SMTPEncryptionImplicitTLS
	}
	return d
}

// Send delivers raw to the envelope's recipients. Once the server accepts the
// message after DATA, Send succeeds: a failed QUIT is only logged, so a retry
// does not deliver the mail twice.
func (d *SMTPDriver) Send(ctx context.Context, envelope Envelope, raw []byte) error {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	conn, err := d.dial(ctx)
	if err != nil {
		return fmt.Errorf("dial smtp server failed: %w", err)
	}
	defer func() { _ = conn.Close() }()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	if err = d.sendMail(ctx, conn, envelope, raw); err != nil {
		return fmt.Errorf("send smtp mail failed: %w", errors.Join(err, ctx.Err()))
	}
	return nil
}

func (d *SMTPDriver) dial(ctx context.Context) (net.Conn, error) {
	if d.encryption == SMTPEncryptionImplicitTLS {
		dialer := tls.Dialer{Config: d.tlsConfig}
		return dialer.DialContext(ctx, "tcp", d.addr)
	}
	var dialer net.Dialer
	return dialer.DialContext(ctx, "tcp", d.addr)
}

func (d *SMTPDriver) sendMail(ctx context.Context, conn net.Conn, envelope Envelope, raw []byte) error {
	client, err := smtp.NewClient(conn, d.host)
	if err != nil {
		return fmt.Errorf("create smtp client failed: %w", err)
	}
	defer func() { _ = client.Close() }()
	if err = client.Hello(d.helo); err != nil {
		return fmt.Errorf("greet smtp server failed: %w", err)
	}
	if err = d.startTLS(client); err != nil {
		return err
	}
	if err = d.authenticate(ctx, client); err != nil {
		return err
	}
	if err = client.Mail(envelope.From); err != nil {
		return fmt.Errorf("set sender failed: %w", err)
	}
	for _, recipient := range envelope.Recipients {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("add recipient failed: %w", err)
		}
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("open data failed: %w", err)
	}
	if _, err = writer.Write(raw); err != nil {
		return fmt.Errorf("write data failed: %w", err)
	}
	if err = writer.Close(); err != nil {
		return fmt.Errorf("close data failed: %w", err)
	}
	// The server has accepted the message; failing now would make the caller
	// retry and deliver it twice.
	if err = client.Quit(); err != nil {
		d.log.WarnContext(
			ctx,
			fmt.Sprintf("smtp quit after accepted message failed: %v", err),
			"addr", d.addr,
		)
	}
	return nil
}

func (d *SMTPDriver) startTLS(client *smtp.Client) error {
	if d.encryption == SMTPEncryptionNone || d.encryption == SMTPEncryptionImplicitTLS {
		return nil
	}
	if ok, _ := client.Extension("STARTTLS"); !ok {
		if d.encryption == SMTPEncryptionStartTLS {
			return errors.New("smtp server does not offer starttls")
		}
		return nil
	}
	if err := client.StartTLS(d.tlsConfig); err != nil {
		return fmt.Errorf("start tls failed: %w", err)
	}
	return nil
}

func (d *SMTPDriver) authenticate(ctx context.Context, client *smtp.Client) error {
	if d.username == "" {
		return nil
	}
	ok, params := client.Extension("AUTH")
	if !ok {
		return errors.New("smtp server does not support auth")
	}
	mechanisms := make(map[string]struct{})
	for mechanism := range strings.FieldsSeq(strings.ToUpper(params)) {
		mechanisms[mechanism] = struct{}{}
	}
	_, encrypted := client.TLSConnectionState()
	auth, err := d.createAuth(ctx, mechanisms, encrypted || d.isLocalHost())
	if err != nil {
		return err
	}
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("authenticate failed: %w", err)
	}
	return nil
}

func (d *SMTPDriver) createAuth(ctx context.Context, mechanisms map[string]struct{}, secure bool) (smtp.Auth, error) {
	if d.tokenSource != nil {
		if _, ok := mechanisms[smtpAuthXOAuth2]; !ok {
			return nil, errors.New("smtp server does not offer xoauth2 auth")
		}
		if !secure {
			return nil, errors.New("smtp xoauth2 auth needs an encrypted connection")
		}
		token, err := d.tokenSource(ctx)
		if err != nil {
			return nil, fmt.Errorf("get smtp oauth2 token failed: %w", err)
		}
		return xoauth2Auth{username: d.username, token: token}, nil
	}
	if secure {
		if _, ok := mechanisms[smtpAuthPlain]; ok {
			return smtp.PlainAuth("", d.username, d.password, d.host), nil
		}
		if _, ok := mechanisms[smtpAuthLogin]; ok {
			return loginAuth{username: d.username, password: d.password}, nil
		}
	}
	if _, ok := mechanisms[smtpAuthCRAMMD5]; ok {
		return smtp.CRAMMD5Auth(d.username, d.password), nil
	}
	if !secure {
		return nil, errors.New("smtp server offers no auth usable over an unencrypted connection")
	}
	return nil, errors.New("smtp server offers no supported auth mechanism")
}

// isLocalHost matches the hosts net/smtp lets PLAIN auth use without TLS.
func (d *SMTPDriver) isLocalHost() bool {
	return d.host == "localhost" || d.host == "127.0.0.1" || d.host == "::1"
}
