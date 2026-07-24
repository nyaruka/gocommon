package smtpx_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/smtpx"
	"github.com/stretchr/testify/assert"
)

func TestSendWithRetries(t *testing.T) {
	ctx := context.Background()

	msg := smtpx.NewMessage([]string{"bob@nyaruka.com", "jim@nyaruka.com"}, "Updates", "Have a great weekend", "")
	c := smtpx.NewClient("mail.temba.io", 255, "leah", "pass123", "updates@temba.io")

	// a sender which errors
	sender := smtpx.NewMockSender(
		errors.New("535 5.7.8 Username and Password not accepted"),            // a non-retriable 5xx error
		errors.New("oops can't send"),                                         // a non-retriable error with no code
		errors.New("421 Service not available, closing transmission channel"), // a retriable error
		errors.New("432 4.7.12 A password transition is needed"),              // a retriable error
		nil, // success
		errors.New("450 Requested mail action not taken: mailbox unavailable"),    // a retriable error
		errors.New("451 Requested action aborted: local error in processing"),     // a retriable error
		errors.New("452 Requested action not taken: insufficient system storage"), // too many retriable errors
		nil, // success
	)
	smtpx.SetSender(sender)

	retries := smtpx.NewFixedRetries(time.Millisecond*100, time.Millisecond*100)

	err := smtpx.Send(ctx, c, msg, retries)
	assert.EqualError(t, err, "535 5.7.8 Username and Password not accepted")
	assert.Equal(t, 1, len(sender.Logs()))

	err = smtpx.Send(ctx, c, msg, retries)
	assert.EqualError(t, err, "oops can't send")
	assert.Equal(t, 2, len(sender.Logs()))

	err = smtpx.Send(ctx, c, msg, retries)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(sender.Logs()))

	err = smtpx.Send(ctx, c, msg, retries)
	assert.EqualError(t, err, "452 Requested action not taken: insufficient system storage")
	assert.Equal(t, 8, len(sender.Logs()))

	err = smtpx.Send(ctx, c, msg, retries)
	assert.NoError(t, err)
	assert.Equal(t, 9, len(sender.Logs()))
}

func TestSendWithCancelledContext(t *testing.T) {
	msg := smtpx.NewMessage([]string{"bob@nyaruka.com"}, "Updates", "Have a great weekend", "")
	c := smtpx.NewClient("mail.temba.io", 255, "leah", "pass123", "updates@temba.io")
	retries := smtpx.NewFixedRetries(time.Millisecond*100, time.Millisecond*100)

	// a sender which errors with retriable errors
	sender := smtpx.NewMockSender(
		errors.New("421 Service not available, closing transmission channel"),
		errors.New("421 Service not available, closing transmission channel"),
	)
	smtpx.SetSender(sender)

	// context which expires during the first retry backoff
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	defer cancel()

	// send returns the last send error without further attempts
	err := smtpx.Send(ctx, c, msg, retries)
	assert.EqualError(t, err, "421 Service not available, closing transmission channel")
	assert.Equal(t, 1, len(sender.Logs()))

	// a sender whose context is dead before the first attempt never sends
	sender = smtpx.NewMockSender(errors.New("421 Service not available, closing transmission channel"))
	smtpx.SetSender(sender)

	err = smtpx.Send(ctx, c, msg, retries)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, 0, len(sender.Logs()))
}
