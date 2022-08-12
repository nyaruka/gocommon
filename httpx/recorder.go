package httpx

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/nyaruka/gocommon/dates"

	"github.com/go-chi/chi/middleware"
	"github.com/pkg/errors"
)

// Recorder is a utility for creating traces of HTTP requests being handled
type Recorder struct {
	Request        *http.Request
	ResponseWriter http.ResponseWriter

	startTime    time.Time
	responseBody *bytes.Buffer

	requestTrace []byte
}

// NewRecorder creates a new recorder for an HTTP request
func NewRecorder(r *http.Request, w http.ResponseWriter) *Recorder {
	responseBody := &bytes.Buffer{}
	wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
	wrapped.Tee(responseBody)

	return &Recorder{
		Request:        r,
		ResponseWriter: wrapped,
		startTime:      dates.Now(),
		responseBody:   responseBody,
	}
}

// SaveRequest immediately saves the request and body for later use. This can be called to guarantee the body
// is available at a later time even if downstream users of the request do not clone the body
func (r *Recorder) SaveRequest() error {
	body, err := io.ReadAll(r.Request.Body)
	if err != nil {
		return errors.Wrapf(err, "error reading body from request")
	}
	r.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	r.requestTrace, err = httputil.DumpRequest(r.Request, true)
	if err != nil {
		return errors.Wrapf(err, "error dumping request")
	}
	r.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	return nil
}

// End is called when the response has been written and generates the trace
func (r *Recorder) End() (*Trace, error) {
	requestTrace := r.requestTrace
	if requestTrace == nil {
		trace, err := httputil.DumpRequest(r.Request, true)
		if err != nil {
			return nil, errors.Wrap(err, "error dumping request")
		}
		requestTrace = trace
	}

	wrapped := r.ResponseWriter.(middleware.WrapResponseWriter)

	// build an approximation of headers part
	responseTrace := &bytes.Buffer{}
	responseTrace.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\r\n", wrapped.Status(), http.StatusText(wrapped.Status())))
	r.ResponseWriter.Header().Write(responseTrace)
	responseTrace.WriteString("\r\n")

	// and parse as response object
	response, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(responseTrace.Bytes())), r.Request)
	if err != nil {
		return nil, errors.Wrap(err, "error reading response trace")
	}

	return &Trace{
		Request:       r.Request,
		RequestTrace:  requestTrace,
		Response:      response,
		ResponseTrace: responseTrace.Bytes(),
		ResponseBody:  r.responseBody.Bytes(),
		StartTime:     r.startTime,
		EndTime:       dates.Now(),
	}, nil
}
