package brevo

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	netmail "net/mail"
	"net/textproto"
	"strings"

	"github.com/daarxwalker/enjin/mail"
)

var (
	ErrEmbedsUnsupported = errors.New("brevo api does not support embedded files")
	ErrMultipleReplyTo   = errors.New("brevo api supports only one reply-to address")
)

type parser struct{}

func (p parser) parse(envelope mail.Envelope, raw []byte) (emailAPIRow, error) {
	var (
		zero  emailAPIRow
		email emailAPIRow
		err   error
	)
	message, err := netmail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return zero, fmt.Errorf("read message failed: %w", err)
	}
	from, err := netmail.ParseAddress(message.Header.Get("From"))
	if err != nil {
		return zero, fmt.Errorf("parse from address failed: %w", err)
	}
	email.Sender = contactAPIRow{Email: from.Address, Name: from.Name}
	if email.To, err = p.parseContacts(message.Header, "To"); err != nil {
		return zero, err
	}
	if email.Cc, err = p.parseContacts(message.Header, "Cc"); err != nil {
		return zero, err
	}
	replyTo, err := p.parseContacts(message.Header, "Reply-To")
	if err != nil {
		return zero, err
	}
	if len(replyTo) > 1 {
		return zero, ErrMultipleReplyTo
	}
	if len(replyTo) == 1 {
		email.ReplyTo = replyTo[0]
	}
	email.Bcc = p.createBcc(envelope, email.To, email.Cc)
	if email.Subject, err = new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject")); err != nil {
		return zero, fmt.Errorf("decode subject failed: %w", err)
	}
	if email, err = p.readPart(textproto.MIMEHeader(message.Header), message.Body, email); err != nil {
		return zero, err
	}
	return email, nil
}

func (p parser) parseContacts(header netmail.Header, key string) ([]contactAPIRow, error) {
	if header.Get(key) == "" {
		return nil, nil
	}
	addresses, err := header.AddressList(key)
	if err != nil {
		return nil, fmt.Errorf("parse %s header failed: %w", key, err)
	}
	contacts := make([]contactAPIRow, 0, len(addresses))
	for _, address := range addresses {
		contacts = append(contacts, contactAPIRow{Email: address.Address, Name: address.Name})
	}
	return contacts, nil
}

// createBcc returns the envelope recipients missing from the To and Cc
// headers; the composed message carries no Bcc header.
func (p parser) createBcc(envelope mail.Envelope, to, cc []contactAPIRow) []contactAPIRow {
	seen := make(map[string]struct{}, len(to)+len(cc))
	for _, contact := range to {
		seen[strings.ToLower(contact.Email)] = struct{}{}
	}
	for _, contact := range cc {
		seen[strings.ToLower(contact.Email)] = struct{}{}
	}
	var bcc []contactAPIRow
	for _, recipient := range envelope.Recipients {
		key := strings.ToLower(recipient)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		bcc = append(bcc, contactAPIRow{Email: recipient})
	}
	return bcc
}

func (p parser) readPart(header textproto.MIMEHeader, body io.Reader, email emailAPIRow) (emailAPIRow, error) {
	var zero emailAPIRow
	mediaType, params, err := mime.ParseMediaType(header.Get("Content-Type"))
	if err != nil {
		return zero, fmt.Errorf("parse content type failed: %w", err)
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		return p.readMultipart(params["boundary"], body, email)
	}
	if header.Get("Content-ID") != "" {
		return zero, ErrEmbedsUnsupported
	}
	content, err := p.decodeBody(header.Get("Content-Transfer-Encoding"), body)
	if err != nil {
		return zero, fmt.Errorf("decode %s body failed: %w", mediaType, err)
	}
	disposition, dispositionParams, err := p.parseDisposition(header.Get("Content-Disposition"))
	if err != nil {
		return zero, err
	}
	switch {
	case disposition == "attachment":
		email.Attachments = append(
			email.Attachments,
			attachmentAPIRow{Name: dispositionParams["filename"], Content: base64.StdEncoding.EncodeToString(content)},
		)
	case mediaType == "text/html":
		email.HTMLContent = string(content)
	case mediaType == "text/plain":
		email.TextContent = string(content)
	default:
		return zero, fmt.Errorf("unsupported part %s", mediaType)
	}
	return email, nil
}

func (p parser) readMultipart(boundary string, body io.Reader, email emailAPIRow) (emailAPIRow, error) {
	var zero emailAPIRow
	if boundary == "" {
		return zero, errors.New("multipart boundary is missing")
	}
	reader := multipart.NewReader(body, boundary)
	for {
		child, err := reader.NextRawPart()
		if errors.Is(err, io.EOF) {
			return email, nil
		}
		if err != nil {
			return zero, fmt.Errorf("read multipart part failed: %w", err)
		}
		if email, err = p.readPart(child.Header, child, email); err != nil {
			return zero, err
		}
	}
}

func (p parser) parseDisposition(value string) (string, map[string]string, error) {
	if value == "" {
		return "", nil, nil
	}
	disposition, params, err := mime.ParseMediaType(value)
	if err != nil {
		return "", nil, fmt.Errorf("parse content disposition failed: %w", err)
	}
	return disposition, params, nil
}

func (p parser) decodeBody(encoding string, body io.Reader) ([]byte, error) {
	switch strings.ToLower(encoding) {
	case "quoted-printable":
		return io.ReadAll(quotedprintable.NewReader(body))
	case "base64":
		return io.ReadAll(base64.NewDecoder(base64.StdEncoding, body))
	default:
		return io.ReadAll(body)
	}
}
