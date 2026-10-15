package brevo_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/daarxwalker/enjin/mail"
	"github.com/daarxwalker/enjin/mail/brevo"

	"github.com/stretchr/testify/require"
)

type brevoTestServer struct {
	server   *httptest.Server
	header   http.Header
	payload  map[string]any
	response string
	status   int
	mu       sync.Mutex
}

func TestDriver(t *testing.T) {
	t.Run(
		"sends the message as api json", func(t *testing.T) {
			srv := newBrevoTestServer(t, http.StatusCreated, `{"messageId":"<1@relay>"}`)
			mailer := mail.NewManager(brevo.NewDriver("secret", brevo.WithEndpoint(srv.server.URL)))
			err := mailer.Send(
				t.Context(),
				mail.Message{
					From:        "Příklad <info@example.com>",
					To:          []string{"Jan Novák <to@example.com>"},
					Cc:          []string{"cc@example.com"},
					Bcc:         []string{"bcc@example.com", "TO@example.com"},
					ReplyTo:     []string{"reply@example.com"},
					Subject:     "Žluťoučký kůň",
					HTML:        "<p>Ahoj světe</p>",
					Text:        "Ahoj světe",
					Attachments: []mail.File{{Name: "faktura.pdf", ContentType: "application/pdf", Bytes: []byte("%PDF-1.7")}},
				},
			)
			require.NoError(t, err)
			srv.mu.Lock()
			defer srv.mu.Unlock()
			require.Equal(t, "secret", srv.header.Get("Api-Key"))
			require.Equal(t, "application/json", srv.header.Get("Content-Type"))
			require.Equal(
				t,
				map[string]any{
					"sender":      map[string]any{"email": "info@example.com", "name": "Příklad"},
					"replyTo":     map[string]any{"email": "reply@example.com"},
					"subject":     "Žluťoučký kůň",
					"htmlContent": "<p>Ahoj světe</p>",
					"textContent": "Ahoj světe",
					"to":          []any{map[string]any{"email": "to@example.com", "name": "Jan Novák"}},
					"cc":          []any{map[string]any{"email": "cc@example.com"}},
					"bcc":         []any{map[string]any{"email": "bcc@example.com"}},
					"attachment":  []any{map[string]any{"name": "faktura.pdf", "content": "JVBERi0xLjc="}},
				},
				srv.payload,
			)
		},
	)

	t.Run(
		"sends a text only message", func(t *testing.T) {
			srv := newBrevoTestServer(t, http.StatusCreated, `{}`)
			mailer := mail.NewManager(brevo.NewDriver("secret", brevo.WithEndpoint(srv.server.URL)))
			err := mailer.Send(
				t.Context(),
				mail.Message{From: "info@example.com", To: []string{"to@example.com"}, Subject: "Hi", Text: "body"},
			)
			require.NoError(t, err)
			srv.mu.Lock()
			defer srv.mu.Unlock()
			require.Equal(t, "body", srv.payload["textContent"])
			require.NotContains(t, srv.payload, "htmlContent")
			require.NotContains(t, srv.payload, "replyTo")
		},
	)

	t.Run(
		"rejects embeds", func(t *testing.T) {
			srv := newBrevoTestServer(t, http.StatusCreated, `{}`)
			mailer := mail.NewManager(brevo.NewDriver("secret", brevo.WithEndpoint(srv.server.URL)))
			err := mailer.Send(
				t.Context(),
				mail.Message{
					From:    "info@example.com",
					To:      []string{"to@example.com"},
					Subject: "Hi",
					HTML:    `<img src="cid:logo">`,
					Embeds:  []mail.File{{Name: "logo", ContentType: "image/png", Bytes: []byte("png")}},
				},
			)
			require.ErrorIs(t, err, brevo.ErrEmbedsUnsupported)
			srv.mu.Lock()
			defer srv.mu.Unlock()
			require.Nil(t, srv.payload)
		},
	)

	t.Run(
		"rejects several reply-to addresses", func(t *testing.T) {
			srv := newBrevoTestServer(t, http.StatusCreated, `{}`)
			mailer := mail.NewManager(brevo.NewDriver("secret", brevo.WithEndpoint(srv.server.URL)))
			err := mailer.Send(
				t.Context(),
				mail.Message{
					From:    "info@example.com",
					To:      []string{"to@example.com"},
					ReplyTo: []string{"a@example.com", "b@example.com"},
					Subject: "Hi",
					Text:    "body",
				},
			)
			require.ErrorIs(t, err, brevo.ErrMultipleReplyTo)
		},
	)

	t.Run(
		"returns the api error", func(t *testing.T) {
			srv := newBrevoTestServer(t, http.StatusBadRequest, `{"code":"invalid_parameter","message":"sender is not valid"}`)
			mailer := mail.NewManager(brevo.NewDriver("secret", brevo.WithEndpoint(srv.server.URL)))
			err := mailer.Send(
				t.Context(),
				mail.Message{From: "info@example.com", To: []string{"to@example.com"}, Subject: "Hi", Text: "body"},
			)
			require.ErrorContains(t, err, "400 Bad Request: invalid_parameter: sender is not valid")
		},
	)
}

func newBrevoTestServer(t *testing.T, status int, response string) *brevoTestServer {
	t.Helper()
	srv := &brevoTestServer{status: status, response: response}
	srv.server = httptest.NewServer(http.HandlerFunc(srv.handle))
	t.Cleanup(srv.server.Close)
	return srv
}

func (s *brevoTestServer) handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var payload map[string]any
	if err = json.Unmarshal(body, &payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	s.header = r.Header.Clone()
	s.payload = payload
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s.status)
	_, _ = io.WriteString(w, s.response)
}
