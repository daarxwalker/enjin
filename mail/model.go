package mail

import "net/textproto"

type Message struct {
	From        string
	To          []string
	Cc          []string
	Bcc         []string
	ReplyTo     []string
	Subject     string
	HTML        string
	Text        string
	Embeds      []File
	Attachments []File
}

type File struct {
	Name        string
	ContentType string
	Bytes       []byte
}

type Envelope struct {
	From       string
	Recipients []string
}

type part struct {
	header   textproto.MIMEHeader
	body     []byte
	subtype  string
	children []part
}
