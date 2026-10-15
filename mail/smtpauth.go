package mail

import (
	"fmt"
	"net/smtp"
	"strings"
)

// loginAuth implements AUTH LOGIN, the only mechanism some servers offer
// (Office 365). The driver uses it only over TLS or to localhost.
type loginAuth struct {
	username string
	password string
}

func (a loginAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return smtpAuthLogin, nil, nil
}

func (a loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	prompt := strings.ToLower(string(fromServer))
	switch {
	case strings.Contains(prompt, "user"):
		return []byte(a.username), nil
	case strings.Contains(prompt, "pass"):
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("unexpected smtp login prompt %q", fromServer)
	}
}

// xoauth2Auth implements AUTH XOAUTH2 with a bearer token (Google Workspace,
// Office 365).
type xoauth2Auth struct {
	username string
	token    string
}

func (a xoauth2Auth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	var response strings.Builder
	response.WriteString("user=")
	response.WriteString(a.username)
	response.WriteString("\x01auth=Bearer ")
	response.WriteString(a.token)
	response.WriteString("\x01\x01")
	return smtpAuthXOAuth2, []byte(response.String()), nil
}

func (a xoauth2Auth) Next(_ []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	// The server rejected the token and sent its reason; an empty answer
	// lets it finish with the final error.
	return []byte{}, nil
}
