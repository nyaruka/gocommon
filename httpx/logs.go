package httpx

import (
	"bytes"
	"regexp"
	"time"

	"github.com/nyaruka/gocommon/stringsx"
)

// LogWithoutTime is a single HTTP trace that can be serialized/deserialized to/from JSON. Note that this struct has no
// time component because it's intended to be embedded in something that does.
type LogWithoutTime struct {
	URL        string `json:"url" validate:"required"`
	StatusCode int    `json:"status_code,omitempty"`
	Request    string `json:"request" validate:"required"`
	Response   string `json:"response,omitempty"`
	ElapsedMS  int    `json:"elapsed_ms"`
	Retries    int    `json:"retries"`
}

// NewLogWithoutTime creates a new log
func NewLogWithoutTime(trace *Trace, trimURLTo, trimTracesTo int, redact stringsx.Redactor) *LogWithoutTime {
	url := trace.Request.URL.String()
	request := string(trace.RequestTrace)
	response := string(ReplaceEscapedNulls(trace.SanitizedResponse("..."), []byte(`�`)))

	statusCode := 0
	if trace.Response != nil {
		statusCode = trace.Response.StatusCode
	}

	if redact != nil {
		url = redact(url)
		request = redact(request)
		response = redact(response)
	}

	return &LogWithoutTime{
		URL:        stringsx.TruncateEllipsis(url, trimURLTo),
		StatusCode: statusCode,
		Request:    stringsx.TruncateEllipsis(request, trimTracesTo),
		Response:   stringsx.TruncateEllipsis(response, trimTracesTo),
		ElapsedMS:  int((trace.EndTime.Sub(trace.StartTime)) / time.Millisecond),
		Retries:    trace.Retries,
	}
}

// Log is a single HTTP trace that can be serialized/deserialized to/from JSON.
type Log struct {
	*LogWithoutTime
	CreatedOn time.Time `json:"created_on" validate:"required"`
}

// NewLog creates a new HTTP log from a trace
func NewLog(trace *Trace, trimURLTo, trimTracesTo int, redact stringsx.Redactor) *Log {
	return &Log{
		NewLogWithoutTime(trace, trimURLTo, trimTracesTo, redact),
		trace.StartTime,
	}
}

// replaces any `\u0000` sequences with the given replacement sequence which may be empty.
// A sequence such as `\\u0000` is preserved as it is an escaped slash followed by the sequence `u0000`
func ReplaceEscapedNulls(data []byte, repl []byte) []byte {
	return nullEscapeRegex.ReplaceAllFunc(data, func(m []byte) []byte {
		slashes := bytes.Count(m, []byte(`\`))
		if slashes%2 == 0 {
			return m
		}

		return append(bytes.Repeat([]byte(`\`), slashes-1), repl...)
	})
}

var nullEscapeRegex = regexp.MustCompile(`\\+u0{4}`)
