package mail

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeDriver struct {
	envelope Envelope
	raw      []byte
}

func (d *fakeDriver) Send(_ context.Context, envelope Envelope, raw []byte) error {
	d.envelope = envelope
	d.raw = raw
	return nil
}

type mimePart struct {
	header   map[string][]string
	body     []byte
	children []mimePart
}

var pngBytes = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{7}, 200)...)

func fullMessage() Message {
	return Message{
		From:        "Žluťoučký Kůň <info@example.com>",
		To:          []string{"Příjemce <to@example.com>"},
		Cc:          []string{"cc@example.com"},
		Bcc:         []string{"hidden@example.com"},
		ReplyTo:     []string{"reply@example.com"},
		Subject:     "Přihlášení do účtu",
		HTML:        `<p>Dobrý den, <img src="cid:logo"></p>`,
		Text:        "Dobrý den,\nklikněte na odkaz.",
		Embeds:      []File{{Name: "logo", Bytes: pngBytes}},
		Attachments: []File{{Name: "faktura č. 1.pdf", ContentType: "application/pdf", Bytes: []byte("%PDF-1.4 obsah")}},
	}
}

func parsePart(t *testing.T, header map[string][]string, body io.Reader) mimePart {
	t.Helper()
	mediaType, params, err := mime.ParseMediaType(header["Content-Type"][0])
	require.NoError(t, err)
	if !strings.HasPrefix(mediaType, "multipart/") {
		raw, readErr := io.ReadAll(body)
		require.NoError(t, readErr)
		return mimePart{header: header, body: raw}
	}
	reader := multipart.NewReader(body, params["boundary"])
	result := mimePart{header: header}
	for {
		child, nextErr := reader.NextRawPart()
		if nextErr == io.EOF {
			break
		}
		require.NoError(t, nextErr)
		result.children = append(result.children, parsePart(t, child.Header, child))
	}
	return result
}

func mediaType(t *testing.T, p mimePart) string {
	t.Helper()
	value, _, err := mime.ParseMediaType(p.header["Content-Type"][0])
	require.NoError(t, err)
	return value
}

func decodeQuotedPrintable(t *testing.T, p mimePart) string {
	t.Helper()
	require.Equal(t, "quoted-printable", p.header["Content-Transfer-Encoding"][0])
	decoded, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(p.body)))
	require.NoError(t, err)
	return string(decoded)
}

func decodeBase64(t *testing.T, p mimePart) []byte {
	t.Helper()
	require.Equal(t, "base64", p.header["Content-Transfer-Encoding"][0])
	for line := range strings.SplitSeq(string(p.body), "\r\n") {
		require.LessOrEqual(t, len(line), base64LineLength)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(string(p.body), "\r\n", ""))
	require.NoError(t, err)
	return decoded
}

func TestComposer(t *testing.T) {
	sentAt := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

	t.Run(
		"full message structure", func(t *testing.T) {
			message := fullMessage()
			raw, err := composer{}.compose(message, sentAt)
			require.NoError(t, err)
			require.NotContains(t, string(raw), "hidden@example.com")
			require.NotContains(t, string(raw), "Bcc")

			parsed, err := mail.ReadMessage(bytes.NewReader(raw))
			require.NoError(t, err)
			from, err := mail.ParseAddress(parsed.Header.Get("From"))
			require.NoError(t, err)
			require.Equal(t, "Žluťoučký Kůň", from.Name)
			require.Equal(t, "info@example.com", from.Address)
			to, err := parsed.Header.AddressList("To")
			require.NoError(t, err)
			require.Equal(t, "Příjemce", to[0].Name)
			require.Equal(t, "<cc@example.com>", parsed.Header.Get("Cc"))
			require.Equal(t, "<reply@example.com>", parsed.Header.Get("Reply-To"))
			subject, err := new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject"))
			require.NoError(t, err)
			require.Equal(t, message.Subject, subject)
			date, err := parsed.Header.Date()
			require.NoError(t, err)
			require.True(t, sentAt.Equal(date))
			require.Regexp(t, `^<[A-Z2-7]+@example\.com>$`, parsed.Header.Get("Message-ID"))
			require.Equal(t, "1.0", parsed.Header.Get("MIME-Version"))

			root := parsePart(t, parsed.Header, parsed.Body)
			require.Equal(t, "multipart/mixed", mediaType(t, root))
			require.Len(t, root.children, 2)

			related := root.children[0]
			require.Equal(t, "multipart/related", mediaType(t, related))
			require.Len(t, related.children, 2)

			alternative := related.children[0]
			require.Equal(t, "multipart/alternative", mediaType(t, alternative))
			require.Len(t, alternative.children, 2)
			require.Equal(t, "text/plain", mediaType(t, alternative.children[0]))
			require.Equal(t, "Dobrý den,\r\nklikněte na odkaz.", decodeQuotedPrintable(t, alternative.children[0]))
			require.Equal(t, "text/html", mediaType(t, alternative.children[1]))
			require.Equal(t, message.HTML, decodeQuotedPrintable(t, alternative.children[1]))

			embed := related.children[1]
			require.Equal(t, "image/png", mediaType(t, embed))
			require.Equal(t, "<logo>", embed.header["Content-Id"][0])
			require.Equal(t, pngBytes, decodeBase64(t, embed))

			attachment := root.children[1]
			require.Equal(t, "application/pdf", mediaType(t, attachment))
			disposition, params, err := mime.ParseMediaType(attachment.header["Content-Disposition"][0])
			require.NoError(t, err)
			require.Equal(t, "attachment", disposition)
			require.Equal(t, "faktura č. 1.pdf", params["filename"])
			require.Equal(t, []byte("%PDF-1.4 obsah"), decodeBase64(t, attachment))
		},
	)

	t.Run(
		"html only message", func(t *testing.T) {
			raw, err := composer{}.compose(Message{From: "info@example.com", To: []string{"to@example.com"}, Subject: "Hi", HTML: "<p>Ahoj</p>"}, sentAt)
			require.NoError(t, err)
			parsed, err := mail.ReadMessage(bytes.NewReader(raw))
			require.NoError(t, err)
			root := parsePart(t, parsed.Header, parsed.Body)
			require.Equal(t, "text/html", mediaType(t, root))
			require.Equal(t, "<p>Ahoj</p>", decodeQuotedPrintable(t, root))
			require.Equal(t, "Hi", parsed.Header.Get("Subject"))
		},
	)

	t.Run(
		"subject cannot inject headers", func(t *testing.T) {
			raw, err := composer{}.compose(Message{From: "info@example.com", To: []string{"to@example.com"}, Subject: "Hi\r\nBcc: evil@example.com", Text: "x"}, sentAt)
			require.NoError(t, err)
			parsed, err := mail.ReadMessage(bytes.NewReader(raw))
			require.NoError(t, err)
			require.Empty(t, parsed.Header.Get("Bcc"))
		},
	)

	t.Run(
		"file fields cannot inject part headers", func(t *testing.T) {
			tests := []struct {
				name    string
				message Message
			}{
				{"embed name", Message{Embeds: []File{{Name: "logo\r\nX-Injected: 1", ContentType: "image/png", Bytes: []byte("png")}}}},
				{"embed name with brackets", Message{Embeds: []File{{Name: "logo>", ContentType: "image/png", Bytes: []byte("png")}}}},
				{"empty embed name", Message{Embeds: []File{{ContentType: "image/png", Bytes: []byte("png")}}}},
				{"embed content type", Message{Embeds: []File{{Name: "logo", ContentType: "image/png\r\nX-Injected: 1", Bytes: []byte("png")}}}},
				{"attachment content type", Message{Attachments: []File{{Name: "a.pdf", ContentType: "application/pdf\r\nX-Injected: 1", Bytes: []byte("pdf")}}}},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					message := tt.message
					message.From = "info@example.com"
					message.To = []string{"to@example.com"}
					message.Text = "x"
					_, err := composer{}.compose(message, sentAt)
					require.Error(t, err)
				})
			}
		},
	)

	t.Run(
		"attachment file name is encoded", func(t *testing.T) {
			raw, err := composer{}.compose(Message{
				From:        "info@example.com",
				To:          []string{"to@example.com"},
				Text:        "x",
				Attachments: []File{{Name: "a.pdf\r\nX-Injected: 1", ContentType: "application/pdf", Bytes: []byte("pdf")}},
			}, sentAt)
			require.NoError(t, err)
			require.NotContains(t, string(raw), "\r\nX-Injected")
		},
	)

	t.Run(
		"reject message without body", func(t *testing.T) {
			_, err := composer{}.compose(Message{From: "info@example.com", To: []string{"to@example.com"}}, sentAt)
			require.Error(t, err)
		},
	)

	t.Run(
		"envelope contains every recipient", func(t *testing.T) {
			envelope, err := composer{}.createEnvelope(fullMessage())
			require.NoError(t, err)
			require.Equal(t, Envelope{
				From:       "info@example.com",
				Recipients: []string{"to@example.com", "cc@example.com", "hidden@example.com"},
			}, envelope)
		},
	)

	t.Run(
		"reject envelope without recipients", func(t *testing.T) {
			_, err := composer{}.createEnvelope(Message{From: "info@example.com", Text: "x"})
			require.Error(t, err)
		},
	)

	t.Run(
		"reject invalid address", func(t *testing.T) {
			_, err := composer{}.createEnvelope(Message{From: "not an address", To: []string{"to@example.com"}})
			require.Error(t, err)
			_, err = composer{}.createEnvelope(Message{From: "info@example.com", Bcc: []string{"broken"}})
			require.Error(t, err)
		},
	)
}

func TestManagerSend(t *testing.T) {
	driver := &fakeDriver{}
	require.NoError(t, NewManager(driver).Send(context.Background(), fullMessage()))
	require.Equal(t, "info@example.com", driver.envelope.From)
	require.Contains(t, driver.envelope.Recipients, "hidden@example.com")
	require.NotContains(t, string(driver.raw), "hidden@example.com")
	_, err := mail.ReadMessage(bytes.NewReader(driver.raw))
	require.NoError(t, err)
}
