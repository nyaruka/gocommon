package smtpx

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/wneessen/go-mail"
)

// Client is an SMTP client
type Client struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// NewClient creates a new client
func NewClient(host string, port int, username, password, from string) *Client {
	return &Client{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

// NewClientFromURL creates a client from a URL like smtp://user:pass@host:port/?from=from@example.com
func NewClientFromURL(connectionURL string) (*Client, error) {
	url, err := url.Parse(connectionURL)
	if err != nil {
		return nil, errors.New("malformed connection URL")
	}
	if url.Scheme != "smtp" {
		return nil, errors.New("connection URL must use SMTP scheme")
	}

	host := url.Hostname()

	// parse port if provided or default to 25
	port := 25
	if url.Port() != "" {
		port, err = strconv.Atoi(url.Port())
		if err != nil || port < 0 || port > 65535 {
			return nil, fmt.Errorf("%s is not a valid port number", url.Port())
		}
	}

	// get the credentials
	if url.User == nil {
		return nil, errors.New("missing credentials in connection URL")
	}
	username := url.User.Username()
	password, _ := url.User.Password()

	// get our from address
	from := url.Query().Get("from")
	if from == "" {
		from = fmt.Sprintf("%s@%s", username, host) // default to username@host if not set
	}

	return NewClient(host, port, username, password, from), nil
}

// Send sends the given message
func (c *Client) Send(ctx context.Context, m *Message) error {
	// create MIME message
	mm := mail.NewMsg()
	if err := mm.From(c.from); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	if err := mm.To(m.recipients...); err != nil {
		return fmt.Errorf("invalid recipient: %w", err)
	}
	mm.Subject(m.subject)
	mm.SetBodyString(mail.TypeTextPlain, m.text)
	if m.html != "" {
		mm.AddAlternativeString(mail.TypeTextHTML, m.html)
	}

	// use implicit SSL/TLS on port 465, otherwise opportunistic STARTTLS
	opts := []mail.Option{mail.WithPort(c.port)}
	if c.port == 465 {
		opts = append(opts, mail.WithSSL())
	} else {
		opts = append(opts, mail.WithTLSPolicy(mail.TLSOpportunistic))
	}
	if c.username != "" {
		opts = append(opts, mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover), mail.WithUsername(c.username), mail.WithPassword(c.password))
	}

	d, err := mail.NewClient(c.host, opts...)
	if err != nil {
		return fmt.Errorf("error creating mail client: %w", err)
	}
	return d.DialAndSendWithContext(ctx, mm)
}
