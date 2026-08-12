// Package svclogs provides a log of an interaction with an external service - the HTTP requests made, any errors
// that occurred, and how long it took. Values which shouldn't be persisted (API keys, tokens) are redacted as they're
// added. Callers typically embed Log in their own type to attach whatever identifies the interaction for them, e.g.
// the channel it was for.
package svclogs

import (
	"time"

	"github.com/nyaruka/gocommon/dates"
	"github.com/nyaruka/gocommon/httpx"
	"github.com/nyaruka/gocommon/stringsx"
	"github.com/nyaruka/gocommon/uuids"
)

// UUID is the type of a service log UUID (should be v7)
type UUID uuids.UUID

// NewUUID creates a new service log UUID
func NewUUID() UUID {
	return UUID(uuids.NewV7())
}

// Type distinguishes the kinds of interaction a log can record, and is defined by the caller
type Type string

// Error is an error that occurred during a service interaction
type Error struct {
	Code    string `json:"code"`
	ExtCode string `json:"ext_code,omitempty"`
	Message string `json:"message"`
}

// Redact applies the given redactor to this error
func (e *Error) Redact(r stringsx.Redactor) *Error {
	return &Error{Code: e.Code, ExtCode: e.ExtCode, Message: r(e.Message)}
}

// Log is the basic service log structure
type Log struct {
	UUID      UUID
	Type      Type
	HttpLogs  []*httpx.Log
	Errors    []*Error
	CreatedOn time.Time
	Elapsed   time.Duration

	recorder *httpx.Recorder
	redactor stringsx.Redactor
}

// New creates a new log of the given type. Pass a recorder when the interaction was triggered by an incoming request
// which should itself be included in the log, and the values (tokens, secrets) to redact from everything added to it.
func New(t Type, r *httpx.Recorder, redactVals []string) *Log {
	return &Log{
		UUID:      NewUUID(),
		Type:      t,
		HttpLogs:  []*httpx.Log{},
		Errors:    []*Error{},
		CreatedOn: dates.Now(),

		recorder: r,
		redactor: stringsx.NewRedactor("**********", redactVals...),
	}
}

// HTTP adds the given HTTP trace to this log
func (l *Log) HTTP(t *httpx.Trace) {
	l.HttpLogs = append(l.HttpLogs, l.traceToLog(t))
}

// Error adds the given error to this log
func (l *Log) Error(e *Error) {
	l.Errors = append(l.Errors, e.Redact(l.redactor))
}

// End finalizes this log
func (l *Log) End() {
	if l.recorder != nil {
		// prepend so it's the first HTTP request in the log
		l.HttpLogs = append([]*httpx.Log{l.traceToLog(l.recorder.Trace)}, l.HttpLogs...)
	}

	l.Elapsed = dates.Now().Sub(l.CreatedOn)
}

// IsError returns whether this log should be considered an error - i.e. we recorded an error or a non 2XX/3XX
// HTTP response
func (l *Log) IsError() bool {
	if len(l.Errors) > 0 {
		return true
	}

	for _, l := range l.HttpLogs {
		if l.StatusCode < 200 || l.StatusCode >= 400 {
			return true
		}
	}

	return false
}

func (l *Log) traceToLog(t *httpx.Trace) *httpx.Log {
	return httpx.NewLog(t, 2048, 50000, l.redactor)
}
