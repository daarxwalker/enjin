package mail

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"time"
)

const smtpTimeout = 30 * time.Second

type SMTPDriver struct {
	auth     smtp.Auth
	addr     string
	host     string
	port     string
	username string
	password string
}

func NewSMTPDriver(
	host string,
	port string,
	username string,
	password string,
) *SMTPDriver {
	d := &SMTPDriver{
		host:     host,
		port:     port,
		addr:     fmt.Sprintf("%s:%s", host, port),
		username: username,
		password: password,
		auth:     smtp.PlainAuth("", username, password, host),
	}
	return d
}

func (d *SMTPDriver) Send(ctx context.Context, envelope Envelope, raw []byte) error {
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", d.addr)
	if err != nil {
		return fmt.Errorf("dial smtp server failed: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if err = conn.SetDeadline(time.Now().Add(smtpTimeout)); err != nil {
		return fmt.Errorf("set smtp deadline failed: %w", err)
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	if err = d.sendMail(conn, envelope, raw); err != nil {
		return fmt.Errorf("send smtp mail failed: %w", errors.Join(err, ctx.Err()))
	}
	return nil
}

func (d *SMTPDriver) sendMail(conn net.Conn, envelope Envelope, raw []byte) error {
	client, err := smtp.NewClient(conn, d.host)
	if err != nil {
		return fmt.Errorf("create smtp client failed: %w", err)
	}
	defer func() { _ = client.Close() }()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err = client.StartTLS(&tls.Config{ServerName: d.host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("start tls failed: %w", err)
		}
	}
	if ok, _ := client.Extension("AUTH"); !ok {
		return errors.New("smtp server does not support auth")
	}
	if err = client.Auth(d.auth); err != nil {
		return fmt.Errorf("authenticate failed: %w", err)
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
	if err = client.Quit(); err != nil {
		return fmt.Errorf("quit failed: %w", err)
	}
	return nil
}
