// Package brevo is the Brevo (formerly Sendinblue) transactional e-mail API
// Driver for package mail.
//
// The API takes structured JSON, not raw MIME, so the Driver reads the message
// composed by mail.Manager back into its parts. Brevo has no inline images and
// only one Reply-To address: a message with embeds or several Reply-To
// addresses is rejected. Send such messages through the Brevo SMTP relay with
// mail.NewSMTPDriver("smtp-relay.brevo.com", "587", login, smtpKey) instead.
package brevo
