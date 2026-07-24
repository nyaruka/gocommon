package smtpx

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wneessen/go-mail"
)

func TestNewClientFromURL(t *testing.T) {
	_, err := NewClientFromURL(":")
	assert.EqualError(t, err, "malformed connection URL")

	_, err = NewClientFromURL("http://")
	assert.EqualError(t, err, "connection URL must use SMTP scheme")

	_, err = NewClientFromURL("smtp://temba.io:1234567890")
	assert.EqualError(t, err, "1234567890 is not a valid port number")

	_, err = NewClientFromURL("smtp://temba.io:25")
	assert.EqualError(t, err, "missing credentials in connection URL")

	// from and port are optional
	s, err := NewClientFromURL("smtp://leah:pass123%21@temba.io")
	assert.NoError(t, err)
	assert.Equal(t, "temba.io", s.host)
	assert.Equal(t, 25, s.port)
	assert.Equal(t, "leah", s.username)
	assert.Equal(t, "pass123!", s.password)
	assert.Equal(t, "leah@temba.io", s.from)

	s, err = NewClientFromURL("smtp://leah%40nyaruka.com:pass123%21@temba.io:255?from=Leah+%3Cupdates%40temba.io%3E")
	assert.NoError(t, err)
	assert.Equal(t, "temba.io", s.host)
	assert.Equal(t, 255, s.port)
	assert.Equal(t, "leah@nyaruka.com", s.username)
	assert.Equal(t, "pass123!", s.password)
	assert.Equal(t, "Leah <updates@temba.io>", s.from)
}

// starts a minimal SMTP server on a random port which rejects MAIL commands with the given response
func newRejectingSMTPServer(t *testing.T, mailResponse string) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { l.Close() })

	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		fmt.Fprint(conn, "220 test ESMTP\r\n")
		br := bufio.NewReader(conn)
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				return
			}
			cmd := strings.ToUpper(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				fmt.Fprint(conn, "250 hello\r\n")
			case strings.HasPrefix(cmd, "MAIL"):
				fmt.Fprint(conn, mailResponse+"\r\n")
			case strings.HasPrefix(cmd, "QUIT"):
				fmt.Fprint(conn, "221 bye\r\n")
				return
			default:
				fmt.Fprint(conn, "250 OK\r\n")
			}
		}
	}()

	return l.Addr().(*net.TCPAddr).Port
}

func TestClientSendErrors(t *testing.T) {
	tcs := []struct {
		mailResponse string
		code         int
		shouldRetry  bool
	}{
		{"421 Service not available, closing transmission channel", 421, true},
		{"550 Requested action not taken: mailbox unavailable", 550, false},
	}

	for _, tc := range tcs {
		port := newRejectingSMTPServer(t, tc.mailResponse)

		c := NewClient("127.0.0.1", port, "", "", "updates@temba.io")
		err := c.Send(context.Background(), NewMessage([]string{"bob@nyaruka.com"}, "Updates", "Hello", ""))
		require.Error(t, err)

		// error should be a go-mail send error wrapping the server response...
		var sendErr *mail.SendError
		require.True(t, errors.As(err, &sendErr))
		assert.Equal(t, tc.code, sendErr.ErrorCode())

		// .. and thus retryable or not based on its code rather than its message
		assert.Equal(t, tc.code, extractCode(err))
		assert.Equal(t, tc.shouldRetry, DefaultShouldRetry(err))
	}
}
