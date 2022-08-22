package httpx

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/go-chi/chi/middleware"
	"github.com/nyaruka/gocommon/dates"
	"github.com/pkg/errors"
)

// Recorder is a utility for creating traces of HTTP requests being handled
type Recorder struct {
	Trace *Trace

	ResponseWriter http.ResponseWriter
	responseBody   *bytes.Buffer
}

// NewRecorder creates a new recorder for an HTTP request. If `originalRequest` is true, it tries to reconstruct the
// original request object.
func NewRecorder(r *http.Request, w http.ResponseWriter, reconstruct bool) (*Recorder, error) {
	or := r
	if reconstruct {
		or = reconstructOriginal(r)
	}

	requestTrace, err := httputil.DumpRequest(or, true)
	if err != nil {
		return nil, errors.Wrap(err, "error dumping request")
	}

	// if we cloned the request above, DumpRequest will have drained the body and saved a copy on the reconstructed
	// request, so put that copy on the passed request as well
	r.Body = or.Body

	responseBody := &bytes.Buffer{}
	wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
	wrapped.Tee(responseBody)

	return &Recorder{
		Trace: &Trace{
			Request:      or,
			RequestTrace: requestTrace,
			StartTime:    dates.Now(),
		},
		ResponseWriter: wrapped,
		responseBody:   responseBody,
	}, nil
}

// End is called when the response has been written and generates the trace
func (r *Recorder) End() error {
	wrapped := r.ResponseWriter.(middleware.WrapResponseWriter)

	// build an approximation of headers part
	responseTrace := &bytes.Buffer{}
	responseTrace.WriteString(fmt.Sprintf("HTTP/1.1 %d %s\r\n", wrapped.Status(), http.StatusText(wrapped.Status())))
	r.ResponseWriter.Header().Write(responseTrace)
	responseTrace.WriteString("\r\n")

	// and parse as response object
	response, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(responseTrace.Bytes())), r.Trace.Request)
	if err != nil {
		return errors.Wrap(err, "error reading response trace")
	}

	r.Trace.Response = response
	r.Trace.ResponseTrace = responseTrace.Bytes()
	r.Trace.ResponseBody = r.responseBody.Bytes()
	r.Trace.EndTime = dates.Now()
	return nil
}

// tries to reconstruct the original client request from the received server request.
func reconstructOriginal(r *http.Request) *http.Request {
	// create copy of request as we'll be modifying the headers and URL
	o := r.Clone(r.Context())
	header := r.Header.Clone()

	host := r.URL.Host
	if host == "" {
		host = r.Host
	}
	if h := r.Header.Get("Host"); h != "" {
		host = h
	}
	if h := r.Header.Get("X-Forwarded-Host"); h != "" {
		host = h
	}

	scheme := r.URL.Scheme
	if scheme == "" {
		scheme = "http"
	}
	if h := r.Header.Get("X-Forwarded-Proto"); h != "" {
		scheme = h
	}

	path := r.RequestURI
	if h := r.Header.Get("X-Forwarded-Path"); h != "" {
		path = h
	}

	for _, h := range stripHeaders {
		header.Del(h)
	}

	// if all that gives us a valid URL, replace it on the request
	u, _ := url.Parse(fmt.Sprintf("%s://%s%s", scheme, host, path))
	if u != nil {
		o.URL = u
		o.RequestURI = path
		o.Header = header
	}

	return o
}

// headers to strip from reconstructed requests (these are nginx and ELB additions)
var stripHeaders = []string{
	"X-Forwarded-Proto",
	"X-Forwarded-Host",
	"X-Forwarded-Port",
	"X-Forwarded-Path",
	"X-Forwarded-For",
	"X-Amzn-Trace-Id",
}
