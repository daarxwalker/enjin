package mail_test

import (
	"bufio"
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/daarxwalker/enjin/mail"

	"github.com/stretchr/testify/require"
)

type smtpTestServer struct {
	listener net.Listener
	commands []string
	data     string
	mu       sync.Mutex
}

func TestSMTPDriver(t *testing.T) {
	t.Run(
		"sends the envelope and data", func(t *testing.T) {
			srv := newSMTPTestServer(t, true)
			driver := mail.NewSMTPDriver("127.0.0.1", srv.port(t), "user", "pass")
			envelope := mail.Envelope{From: "info@example.com", Recipients: []string{"to@example.com", "cc@example.com"}}
			require.NoError(t, driver.Send(t.Context(), envelope, []byte("Subject: Hi\r\n\r\nbody\r\n")))
			srv.mu.Lock()
			defer srv.mu.Unlock()
			require.Contains(t, srv.commands, "MAIL FROM:<info@example.com>")
			require.Contains(t, srv.commands, "RCPT TO:<to@example.com>")
			require.Contains(t, srv.commands, "RCPT TO:<cc@example.com>")
			require.Equal(t, "Subject: Hi\r\n\r\nbody\r\n", srv.data)
		},
	)

	t.Run(
		"canceled context stops a stuck server", func(t *testing.T) {
			srv := newSMTPTestServer(t, false)
			driver := mail.NewSMTPDriver("127.0.0.1", srv.port(t), "user", "pass")
			ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
			defer cancel()
			start := time.Now()
			err := driver.Send(ctx, mail.Envelope{From: "info@example.com", Recipients: []string{"to@example.com"}}, []byte("x"))
			require.ErrorIs(t, err, context.DeadlineExceeded)
			require.Less(t, time.Since(start), 5*time.Second)
		},
	)
}

func newSMTPTestServer(t *testing.T, respond bool) *smtpTestServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := &smtpTestServer{listener: listener}
	t.Cleanup(func() { _ = listener.Close() })
	go srv.accept(respond)
	return srv
}

func (s *smtpTestServer) port(t *testing.T) string {
	t.Helper()
	_, port, err := net.SplitHostPort(s.listener.Addr().String())
	require.NoError(t, err)
	return port
}

func (s *smtpTestServer) accept(respond bool) {
	conn, err := s.listener.Accept()
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()
	if !respond {
		_, _ = bufio.NewReader(conn).ReadString('\n')
		return
	}
	reader := bufio.NewReader(conn)
	_, _ = conn.Write([]byte("220 test\r\n"))
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			return
		}
		command := strings.TrimRight(line, "\r\n")
		s.mu.Lock()
		s.commands = append(s.commands, command)
		s.mu.Unlock()
		switch {
		case strings.HasPrefix(command, "EHLO"):
			_, _ = conn.Write([]byte("250-test\r\n250 AUTH PLAIN\r\n"))
		case strings.HasPrefix(command, "AUTH"):
			_, _ = conn.Write([]byte("235 ok\r\n"))
		case command == "DATA":
			_, _ = conn.Write([]byte("354 go\r\n"))
			s.readData(reader)
			_, _ = conn.Write([]byte("250 ok\r\n"))
		case command == "QUIT":
			_, _ = conn.Write([]byte("221 bye\r\n"))
			return
		default:
			_, _ = conn.Write([]byte("250 ok\r\n"))
		}
	}
}

func (s *smtpTestServer) readData(reader *bufio.Reader) {
	var data strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil || line == ".\r\n" {
			break
		}
		data.WriteString(line)
	}
	s.mu.Lock()
	s.data = data.String()
	s.mu.Unlock()
}
