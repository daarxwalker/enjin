package mail_test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/mail"

	"github.com/stretchr/testify/require"
)

type smtpTestServer struct {
	listener  net.Listener
	config    smtpTestConfig
	commands  []string
	auth      []string
	failures  []error
	data      string
	encrypted bool
	mu        sync.Mutex
}

type smtpTestConfig struct {
	// startTLS offers STARTTLS with this config when set.
	startTLS *tls.Config
	// implicitTLS speaks TLS from the first byte with this config when set.
	implicitTLS *tls.Config
	extensions  []string
	silent      bool
	failQuit    bool
}

var testEnvelope = mail.Envelope{From: "info@example.com", Recipients: []string{"to@example.com"}}

func TestSMTPDriver(t *testing.T) {
	t.Run(
		"sends the envelope and data", func(t *testing.T) {
			srv := newSMTPTestServer(t, smtpTestConfig{extensions: []string{"AUTH PLAIN"}})
			driver := mail.NewSMTPDriver("127.0.0.1", srv.port(t), "user", "pass")
			envelope := mail.Envelope{From: "info@example.com", Recipients: []string{"to@example.com", "cc@example.com"}}
			require.NoError(t, driver.Send(t.Context(), envelope, []byte("Subject: Hi\r\n\r\nbody\r\n")))
			srv.mu.Lock()
			defer srv.mu.Unlock()
			require.Contains(t, srv.commands, "MAIL FROM:<info@example.com>")
			require.Contains(t, srv.commands, "RCPT TO:<to@example.com>")
			require.Contains(t, srv.commands, "RCPT TO:<cc@example.com>")
			require.Equal(t, []string{"\x00user\x00pass"}, srv.auth)
			require.Equal(t, "Subject: Hi\r\n\r\nbody\r\n", srv.data)
		},
	)

	t.Run(
		"canceled context stops a stuck server", func(t *testing.T) {
			srv := newSMTPTestServer(t, smtpTestConfig{silent: true})
			driver := mail.NewSMTPDriver("127.0.0.1", srv.port(t), "user", "pass")
			ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
			defer cancel()
			start := time.Now()
			err := driver.Send(ctx, testEnvelope, []byte("x"))
			require.ErrorIs(t, err, context.DeadlineExceeded)
			require.Less(t, time.Since(start), 5*time.Second)
		},
	)

	t.Run(
		"timeout stops a stuck server", func(t *testing.T) {
			srv := newSMTPTestServer(t, smtpTestConfig{silent: true})
			driver := mail.NewSMTPDriver(
				"127.0.0.1",
				srv.port(t),
				"user",
				"pass",
				mail.WithSMTPTimeout(100*time.Millisecond),
			)
			err := driver.Send(t.Context(), testEnvelope, []byte("x"))
			require.ErrorIs(t, err, context.DeadlineExceeded)
		},
	)

	t.Run(
		"greets with the configured name", func(t *testing.T) {
			srv := newSMTPTestServer(t, smtpTestConfig{})
			driver := mail.NewSMTPDriver("127.0.0.1", srv.port(t), "", "", mail.WithSMTPHelo("mail.example.com"))
			require.NoError(t, driver.Send(t.Context(), testEnvelope, []byte("x")))
			srv.mu.Lock()
			defer srv.mu.Unlock()
			require.Equal(t, "EHLO mail.example.com", srv.commands[0])
		},
	)

	t.Run(
		"skips auth without a username", func(t *testing.T) {
			srv := newSMTPTestServer(t, smtpTestConfig{})
			driver := mail.NewSMTPDriver("127.0.0.1", srv.port(t), "", "")
			require.NoError(t, driver.Send(t.Context(), testEnvelope, []byte("x")))
			srv.mu.Lock()
			defer srv.mu.Unlock()
			require.Empty(t, srv.auth)
			require.Equal(t, "x\r\n", srv.data)
		},
	)

	t.Run(
		"fails with a username when the server has no auth", func(t *testing.T) {
			srv := newSMTPTestServer(t, smtpTestConfig{})
			driver := mail.NewSMTPDriver("127.0.0.1", srv.port(t), "user", "pass")
			require.ErrorContains(t, driver.Send(t.Context(), testEnvelope, []byte("x")), "does not support auth")
		},
	)

	t.Run(
		"uses login when the server offers only login", func(t *testing.T) {
			srv := newSMTPTestServer(t, smtpTestConfig{extensions: []string{"AUTH LOGIN XOAUTH2"}})
			driver := mail.NewSMTPDriver("127.0.0.1", srv.port(t), "user", "pass")
			require.NoError(t, driver.Send(t.Context(), testEnvelope, []byte("x")))
			srv.mu.Lock()
			defer srv.mu.Unlock()
			require.Equal(t, []string{"user", "pass"}, srv.auth)
		},
	)

	t.Run(
		"uses xoauth2 with a fresh token", func(t *testing.T) {
			srv := newSMTPTestServer(t, smtpTestConfig{extensions: []string{"AUTH PLAIN LOGIN XOAUTH2"}})
			driver := mail.NewSMTPDriver(
				"127.0.0.1",
				srv.port(t),
				"user@example.com",
				"",
				mail.WithSMTPOAuth2(func(context.Context) (string, error) {
					return "token", nil
				}),
			)
			require.NoError(t, driver.Send(t.Context(), testEnvelope, []byte("x")))
			srv.mu.Lock()
			defer srv.mu.Unlock()
			require.Equal(t, []string{"user=user@example.com\x01auth=Bearer token\x01\x01"}, srv.auth)
		},
	)

	t.Run(
		"upgrades with starttls when offered", func(t *testing.T) {
			serverTLS, clientTLS := newTestTLSConfigs(t)
			srv := newSMTPTestServer(t, smtpTestConfig{startTLS: serverTLS, extensions: []string{"AUTH PLAIN"}})
			driver := mail.NewSMTPDriver("127.0.0.1", srv.port(t), "user", "pass", mail.WithSMTPTLSConfig(clientTLS))
			require.NoError(t, driver.Send(t.Context(), testEnvelope, []byte("x")))
			srv.mu.Lock()
			defer srv.mu.Unlock()
			require.Contains(t, srv.commands, "STARTTLS")
			require.True(t, srv.encrypted)
		},
	)

	t.Run(
		"required starttls fails when not offered", func(t *testing.T) {
			srv := newSMTPTestServer(t, smtpTestConfig{})
			driver := mail.NewSMTPDriver(
				"127.0.0.1",
				srv.port(t),
				"",
				"",
				mail.WithSMTPEncryption(mail.SMTPEncryptionStartTLS),
			)
			require.ErrorContains(t, driver.Send(t.Context(), testEnvelope, []byte("x")), "does not offer starttls")
			srv.mu.Lock()
			defer srv.mu.Unlock()
			require.NotContains(t, srv.commands, "MAIL FROM:<info@example.com>")
		},
	)

	t.Run(
		"no encryption ignores offered starttls", func(t *testing.T) {
			serverTLS, _ := newTestTLSConfigs(t)
			srv := newSMTPTestServer(t, smtpTestConfig{startTLS: serverTLS})
			driver := mail.NewSMTPDriver(
				"127.0.0.1",
				srv.port(t),
				"",
				"",
				mail.WithSMTPEncryption(mail.SMTPEncryptionNone),
			)
			require.NoError(t, driver.Send(t.Context(), testEnvelope, []byte("x")))
			srv.mu.Lock()
			defer srv.mu.Unlock()
			require.NotContains(t, srv.commands, "STARTTLS")
			require.False(t, srv.encrypted)
		},
	)

	t.Run(
		"speaks implicit tls", func(t *testing.T) {
			serverTLS, clientTLS := newTestTLSConfigs(t)
			srv := newSMTPTestServer(t, smtpTestConfig{implicitTLS: serverTLS, extensions: []string{"AUTH PLAIN"}})
			driver := mail.NewSMTPDriver(
				"127.0.0.1",
				srv.port(t),
				"user",
				"pass",
				mail.WithSMTPEncryption(mail.SMTPEncryptionImplicitTLS),
				mail.WithSMTPTLSConfig(clientTLS),
			)
			require.NoError(t, driver.Send(t.Context(), testEnvelope, []byte("x")))
			srv.mu.Lock()
			defer srv.mu.Unlock()
			require.True(t, srv.encrypted)
			require.Equal(t, "x\r\n", srv.data)
		},
	)

	t.Run(
		"failed quit after accepted data succeeds", func(t *testing.T) {
			var logs bytes.Buffer
			srv := newSMTPTestServer(t, smtpTestConfig{failQuit: true})
			driver := mail.NewSMTPDriver(
				"127.0.0.1",
				srv.port(t),
				"",
				"",
				mail.WithSMTPLogger(slog.New(slog.NewTextHandler(&logs, nil))),
			)
			require.NoError(t, driver.Send(t.Context(), testEnvelope, []byte("x")))
			require.Contains(t, logs.String(), "smtp quit after accepted message failed")
			srv.mu.Lock()
			defer srv.mu.Unlock()
			require.Equal(t, "x\r\n", srv.data)
		},
	)
}

func newSMTPTestServer(t *testing.T, config smtpTestConfig) *smtpTestServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	if config.implicitTLS != nil {
		listener = tls.NewListener(listener, config.implicitTLS)
	}
	srv := &smtpTestServer{listener: listener, config: config}
	t.Cleanup(func() {
		srv.mu.Lock()
		defer srv.mu.Unlock()
		require.NoError(t, errors.Join(srv.failures...))
	})
	t.Cleanup(func() {
		require.NoError(t, listener.Close())
	})
	go srv.accept()
	return srv
}

// newTestTLSConfigs returns a server config with a self-signed certificate
// for 127.0.0.1 and a client config that trusts it.
func newTestTLSConfigs(t *testing.T) (*tls.Config, *tls.Config) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	serverTLS := &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}
	return serverTLS, &tls.Config{RootCAs: pool}
}

func (s *smtpTestServer) port(t *testing.T) string {
	t.Helper()
	_, port, err := net.SplitHostPort(s.listener.Addr().String())
	require.NoError(t, err)
	return port
}

// accept serves one client. The tests hang up early on purpose (timeouts,
// refused auth), so a hang-up only ends the session; any other failure is
// recorded and fails the test in its cleanup.
func (s *smtpTestServer) accept() {
	conn, err := s.listener.Accept()
	if err != nil {
		if !s.isHangUp(err) {
			s.recordFailure(fmt.Errorf("accept test connection failed: %w", err))
		}
		return
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil && !s.isHangUp(closeErr) {
			s.recordFailure(fmt.Errorf("close test connection failed: %w", closeErr))
		}
	}()
	if s.config.silent {
		if _, readErr := bufio.NewReader(conn).ReadString('\n'); readErr != nil && !s.isHangUp(readErr) {
			s.recordFailure(fmt.Errorf("wait for the client to hang up failed: %w", readErr))
		}
		return
	}
	s.serve(conn)
}

func (s *smtpTestServer) reply(conn net.Conn, line string) bool {
	if _, err := conn.Write([]byte(line)); err != nil {
		if !s.isHangUp(err) {
			s.recordFailure(fmt.Errorf("write %q failed: %w", line, err))
		}
		return false
	}
	return true
}

func (s *smtpTestServer) isHangUp(err error) bool {
	return errors.Is(err, io.EOF) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EPIPE)
}

func (s *smtpTestServer) recordFailure(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures = append(s.failures, err)
}

func (s *smtpTestServer) serve(conn net.Conn) {
	reader := bufio.NewReader(conn)
	if !s.reply(conn, "220 test\r\n") {
		return
	}
	for {
		line, ok := s.readLine(reader)
		if !ok {
			return
		}
		command := strings.TrimRight(line, "\r\n")
		s.mu.Lock()
		s.commands = append(s.commands, command)
		s.mu.Unlock()
		var replied bool
		switch {
		case strings.HasPrefix(command, "EHLO"):
			replied = s.reply(conn, s.createEHLOResponse())
		case command == "STARTTLS":
			replied = s.reply(conn, "220 ready\r\n")
			conn = tls.Server(conn, s.config.startTLS)
			reader = bufio.NewReader(conn)
		case command == "AUTH LOGIN":
			replied = s.readLogin(conn, reader)
		case strings.HasPrefix(command, "AUTH "):
			s.recordAuth(strings.Fields(command)[2])
			replied = s.reply(conn, "235 ok\r\n")
		case strings.HasPrefix(command, "MAIL"):
			_, encrypted := conn.(*tls.Conn)
			s.mu.Lock()
			s.encrypted = encrypted
			s.mu.Unlock()
			replied = s.reply(conn, "250 ok\r\n")
		case command == "DATA":
			replied = s.reply(conn, "354 go\r\n") && s.readData(reader) && s.reply(conn, "250 ok\r\n")
		case command == "QUIT":
			if !s.config.failQuit {
				s.reply(conn, "221 bye\r\n")
			}
			return
		default:
			replied = s.reply(conn, "250 ok\r\n")
		}
		if !replied {
			return
		}
	}
}

func (s *smtpTestServer) createEHLOResponse() string {
	var response strings.Builder
	response.WriteString("250-test\r\n")
	for _, extension := range s.config.extensions {
		response.WriteString("250-")
		response.WriteString(extension)
		response.WriteString("\r\n")
	}
	if s.config.startTLS != nil {
		response.WriteString("250-STARTTLS\r\n")
	}
	response.WriteString("250 SIZE 10485760\r\n")
	return response.String()
}

func (s *smtpTestServer) readLogin(conn net.Conn, reader *bufio.Reader) bool {
	for _, prompt := range []string{"Username:", "Password:"} {
		if !s.reply(conn, "334 "+base64.StdEncoding.EncodeToString([]byte(prompt))+"\r\n") {
			return false
		}
		line, ok := s.readLine(reader)
		if !ok {
			return false
		}
		s.recordAuth(strings.TrimRight(line, "\r\n"))
	}
	return s.reply(conn, "235 ok\r\n")
}

func (s *smtpTestServer) recordAuth(encoded string) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		s.recordFailure(fmt.Errorf("decode auth %q failed: %w", encoded, err))
		return
	}
	s.mu.Lock()
	s.auth = append(s.auth, string(decoded))
	s.mu.Unlock()
}

func (s *smtpTestServer) readData(reader *bufio.Reader) bool {
	var data strings.Builder
	for {
		line, ok := s.readLine(reader)
		if !ok {
			return false
		}
		if line == ".\r\n" {
			break
		}
		data.WriteString(line)
	}
	s.mu.Lock()
	s.data = data.String()
	s.mu.Unlock()
	return true
}

func (s *smtpTestServer) readLine(reader *bufio.Reader) (string, bool) {
	line, err := reader.ReadString('\n')
	if err != nil {
		if !s.isHangUp(err) {
			s.recordFailure(fmt.Errorf("read command failed: %w", err))
		}
		return "", false
	}
	return line, true
}
