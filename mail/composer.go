package mail

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/http"
	"net/mail"
	"net/textproto"
	"slices"
	"strings"
	"time"
	"unicode"
)

const base64LineLength = 76

type composer struct{}

func (c composer) createEnvelope(message Message) (Envelope, error) {
	var zero Envelope
	from, err := mail.ParseAddress(message.From)
	if err != nil {
		return zero, fmt.Errorf("parse from address failed: %w", err)
	}
	recipients := make([]string, 0, len(message.To)+len(message.Cc)+len(message.Bcc))
	for _, recipient := range slices.Concat(message.To, message.Cc, message.Bcc) {
		address, parseErr := mail.ParseAddress(recipient)
		if parseErr != nil {
			return zero, fmt.Errorf("parse recipient %q failed: %w", recipient, parseErr)
		}
		recipients = append(recipients, address.Address)
	}
	if len(recipients) == 0 {
		return zero, errors.New("message has no recipients")
	}
	return Envelope{From: from.Address, Recipients: recipients}, nil
}

func (c composer) compose(message Message, sentAt time.Time) ([]byte, error) {
	from, err := mail.ParseAddress(message.From)
	if err != nil {
		return nil, fmt.Errorf("parse from address failed: %w", err)
	}
	root, err := c.createRoot(message)
	if err != nil {
		return nil, err
	}
	rootHeader, rootBody, err := c.render(root)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	c.writeHeader(&buf, "From", from.String())
	for _, field := range []struct {
		key       string
		addresses []string
	}{
		{key: "Reply-To", addresses: message.ReplyTo},
		{key: "To", addresses: message.To},
		{key: "Cc", addresses: message.Cc},
	} {
		if len(field.addresses) == 0 {
			continue
		}
		value, formatErr := c.formatAddressList(field.addresses)
		if formatErr != nil {
			return nil, fmt.Errorf("format %s header failed: %w", field.key, formatErr)
		}
		c.writeHeader(&buf, field.key, value)
	}
	c.writeHeader(&buf, "Subject", mime.BEncoding.Encode("UTF-8", message.Subject))
	c.writeHeader(&buf, "Date", sentAt.Format(time.RFC1123Z))
	c.writeHeader(&buf, "Message-ID", c.createMessageID(from.Address))
	c.writeHeader(&buf, "MIME-Version", "1.0")
	keys := make([]string, 0, len(rootHeader))
	for key := range rootHeader {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		c.writeHeader(&buf, key, rootHeader.Get(key))
	}
	buf.WriteString("\r\n")
	buf.Write(rootBody)
	return buf.Bytes(), nil
}

func (c composer) createRoot(message Message) (part, error) {
	var body part
	switch {
	case message.HTML != "" && message.Text != "":
		textPart, err := c.createTextPart("text/plain", message.Text)
		if err != nil {
			return part{}, err
		}
		htmlPart, err := c.createTextPart("text/html", message.HTML)
		if err != nil {
			return part{}, err
		}
		body = part{subtype: "alternative", children: []part{textPart, htmlPart}}
	case message.HTML != "":
		htmlPart, err := c.createTextPart("text/html", message.HTML)
		if err != nil {
			return part{}, err
		}
		body = htmlPart
	case message.Text != "":
		textPart, err := c.createTextPart("text/plain", message.Text)
		if err != nil {
			return part{}, err
		}
		body = textPart
	default:
		return part{}, errors.New("message has no body")
	}
	if len(message.Embeds) > 0 {
		children := []part{body}
		for _, embed := range message.Embeds {
			embedPart, err := c.createEmbedPart(embed)
			if err != nil {
				return part{}, err
			}
			children = append(children, embedPart)
		}
		body = part{subtype: "related", children: children}
	}
	if len(message.Attachments) > 0 {
		children := []part{body}
		for _, attachment := range message.Attachments {
			attachmentPart, err := c.createAttachmentPart(attachment)
			if err != nil {
				return part{}, err
			}
			children = append(children, attachmentPart)
		}
		body = part{subtype: "mixed", children: children}
	}
	return body, nil
}

func (c composer) createTextPart(contentType, text string) (part, error) {
	var buf bytes.Buffer
	writer := quotedprintable.NewWriter(&buf)
	if _, err := writer.Write([]byte(text)); err != nil {
		return part{}, fmt.Errorf("encode %s body failed: %w", contentType, err)
	}
	if err := writer.Close(); err != nil {
		return part{}, fmt.Errorf("encode %s body failed: %w", contentType, err)
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", mime.FormatMediaType(contentType, map[string]string{"charset": "UTF-8"}))
	header.Set("Content-Transfer-Encoding", "quoted-printable")
	return part{header: header, body: buf.Bytes()}, nil
}

func (c composer) createEmbedPart(embed File) (part, error) {
	if !c.isContentID(embed.Name) {
		return part{}, fmt.Errorf("embed name %q is not a valid content id", embed.Name)
	}
	header, err := c.createFileHeader(embed, "inline")
	if err != nil {
		return part{}, err
	}
	header.Set("Content-ID", "<"+embed.Name+">")
	return part{header: header, body: c.encodeBase64(embed.Bytes)}, nil
}

func (c composer) createAttachmentPart(attachment File) (part, error) {
	header, err := c.createFileHeader(attachment, "attachment")
	if err != nil {
		return part{}, err
	}
	return part{header: header, body: c.encodeBase64(attachment.Bytes)}, nil
}

func (c composer) createFileHeader(file File, disposition string) (textproto.MIMEHeader, error) {
	contentType := file.ContentType
	if contentType == "" {
		contentType = http.DetectContentType(file.Bytes)
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("parse content type of %q failed: %w", file.Name, err)
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", mime.FormatMediaType(mediaType, params))
	header.Set("Content-Transfer-Encoding", "base64")
	header.Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": file.Name}))
	return header, nil
}

func (c composer) isContentID(name string) bool {
	if name == "" {
		return false
	}
	return !strings.ContainsFunc(name, func(char rune) bool {
		return char == '<' || char == '>' || unicode.IsSpace(char) || unicode.IsControl(char)
	})
}

func (c composer) createMessageID(fromAddress string) string {
	domain := fromAddress[strings.LastIndex(fromAddress, "@")+1:]
	return "<" + rand.Text() + "@" + domain + ">"
}

func (c composer) render(p part) (textproto.MIMEHeader, []byte, error) {
	if len(p.children) == 0 {
		return p.header, p.body, nil
	}
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for _, child := range p.children {
		childHeader, childBody, err := c.render(child)
		if err != nil {
			return nil, nil, err
		}
		partWriter, err := writer.CreatePart(childHeader)
		if err != nil {
			return nil, nil, fmt.Errorf("create %s part failed: %w", p.subtype, err)
		}
		if _, err = partWriter.Write(childBody); err != nil {
			return nil, nil, fmt.Errorf("write %s part failed: %w", p.subtype, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, nil, fmt.Errorf("close %s part failed: %w", p.subtype, err)
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", mime.FormatMediaType("multipart/"+p.subtype, map[string]string{"boundary": writer.Boundary()}))
	return header, buf.Bytes(), nil
}

func (c composer) formatAddressList(addresses []string) (string, error) {
	formatted := make([]string, 0, len(addresses))
	for _, value := range addresses {
		address, err := mail.ParseAddress(value)
		if err != nil {
			return "", fmt.Errorf("parse address %q failed: %w", value, err)
		}
		formatted = append(formatted, address.String())
	}
	return strings.Join(formatted, ", "), nil
}

func (c composer) encodeBase64(data []byte) []byte {
	encoded := base64.StdEncoding.EncodeToString(data)
	var buf bytes.Buffer
	for len(encoded) > base64LineLength {
		buf.WriteString(encoded[:base64LineLength])
		buf.WriteString("\r\n")
		encoded = encoded[base64LineLength:]
	}
	buf.WriteString(encoded)
	return buf.Bytes()
}

func (c composer) writeHeader(buf *bytes.Buffer, key, value string) {
	buf.WriteString(key + ": " + value + "\r\n")
}
