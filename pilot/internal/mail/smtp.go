package mail

import (
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// Client sends emails via Mailpit's SMTP interface.
type Client struct {
	Host string
	Port string
}

// NewClient creates an SMTP client targeting Mailpit.
func NewClient(host, port string) *Client {
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "1025"
	}
	return &Client{Host: host, Port: port}
}

// Message represents an email to send.
type Message struct {
	From    string
	To      []string
	Subject string
	Text    string
	HTML    string
	ReplyTo string
}

// Send delivers a message via SMTP.
func (c *Client) Send(msg *Message) error {
	if msg.From == "" || len(msg.To) == 0 {
		return fmt.Errorf("from and to are required")
	}

	addr := net.JoinHostPort(c.Host, c.Port)

	body := buildMessage(msg)

	return smtp.SendMail(addr, nil, msg.From, msg.To, []byte(body))
}

// stripCRLF removes CR and LF bytes from a header value to prevent
// SMTP header injection (CWE-93). Without this, an attacker who controls
// any header field can inject additional headers like Bcc:.
func stripCRLF(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}

func buildMessage(msg *Message) string {
	var b strings.Builder

	b.WriteString("From: " + stripCRLF(msg.From) + "\r\n")
	to := make([]string, len(msg.To))
	for i, addr := range msg.To {
		to[i] = stripCRLF(addr)
	}
	b.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	b.WriteString("Subject: " + stripCRLF(msg.Subject) + "\r\n")
	if msg.ReplyTo != "" {
		b.WriteString("Reply-To: " + stripCRLF(msg.ReplyTo) + "\r\n")
	}
	b.WriteString("MIME-Version: 1.0\r\n")

	if msg.HTML != "" {
		boundary := "----=_PostpilotBoundary"
		b.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n")
		b.WriteString("\r\n")

		// Text part
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
		text := msg.Text
		if text == "" {
			text = "Please view this email in an HTML-capable client."
		}
		b.WriteString(text + "\r\n")

		// HTML part
		b.WriteString("--" + boundary + "\r\n")
		b.WriteString("Content-Type: text/html; charset=utf-8\r\n\r\n")
		b.WriteString(msg.HTML + "\r\n")

		b.WriteString("--" + boundary + "--\r\n")
	} else {
		b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
		b.WriteString("\r\n")
		b.WriteString(msg.Text + "\r\n")
	}

	return b.String()
}
