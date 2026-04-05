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

func buildMessage(msg *Message) string {
	var b strings.Builder

	b.WriteString("From: " + msg.From + "\r\n")
	b.WriteString("To: " + strings.Join(msg.To, ", ") + "\r\n")
	b.WriteString("Subject: " + msg.Subject + "\r\n")
	if msg.ReplyTo != "" {
		b.WriteString("Reply-To: " + msg.ReplyTo + "\r\n")
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
