package brevo

type emailAPIRow struct {
	Sender      contactAPIRow      `json:"sender"`
	ReplyTo     contactAPIRow      `json:"replyTo,omitzero"`
	Subject     string             `json:"subject"`
	HTMLContent string             `json:"htmlContent,omitempty"`
	TextContent string             `json:"textContent,omitempty"`
	To          []contactAPIRow    `json:"to"`
	Cc          []contactAPIRow    `json:"cc,omitempty"`
	Bcc         []contactAPIRow    `json:"bcc,omitempty"`
	Attachments []attachmentAPIRow `json:"attachment,omitempty"`
}

type contactAPIRow struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type attachmentAPIRow struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type errorAPIRow struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
