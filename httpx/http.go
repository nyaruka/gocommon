package httpx

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gabriel-vasile/mimetype"
)

// ErrResponseSize is returned when response size exceeds provided limit
var ErrResponseSize = errors.New("response body exceeds size limit")

// ErrAccessConfig is returned when provided access config prevents request
var ErrAccessConfig = errors.New("request not permitted by access config")

// Do makes the given HTTP request using the current requestor and retry config
//
// Deprecated: Do bundles request concerns (retrying, access control) that are better handled by composing
// http.RoundTripper wrappers; it will be removed in a future release.
func Do(client *http.Client, request *http.Request, retries *RetryConfig, access *AccessConfig) (*http.Response, error) {
	r, _, err := do(client, request, retries, access)
	return r, err
}

func do(client *http.Client, request *http.Request, retries *RetryConfig, access *AccessConfig) (*http.Response, int, error) {
	if err := access.check(request); err != nil {
		return nil, 0, err
	}

	var response *http.Response
	var err error
	retry := 0

	for {
		response, err = currentRequestor.Do(client, request)

		if retries != nil && retry < retries.MaxRetries() {
			backoff := retries.Backoff(retry)

			if retries.ShouldRetry(request, response, backoff) {
				time.Sleep(backoff)
				retry++
				continue
			}
		}

		break
	}

	return response, retry, err
}

// NewRequest is a convenience method to create a request with the given context and headers
func NewRequest(ctx context.Context, method string, url string, body io.Reader, headers map[string]string) (*http.Request, error) {
	r, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		r.Header.Set(key, value)
	}

	return r, nil
}

// Requestor is anything that can make an HTTP request with a client
type Requestor interface {
	Do(*http.Client, *http.Request) (*http.Response, error)
}

type defaultRequestor struct{}

func (r defaultRequestor) Do(client *http.Client, request *http.Request) (*http.Response, error) {
	return client.Do(request)
}

// DefaultRequestor is the default HTTP requestor
var DefaultRequestor Requestor = defaultRequestor{}
var currentRequestor = DefaultRequestor

// SetRequestor sets the requestor used by Request
func SetRequestor(requestor Requestor) {
	currentRequestor = requestor
}

// DetectContentType is a replacement for http.DetectContentType which leans on the github.com/gabriel-vasile/mimetype
// library to support more types, and additionally returns the extension (including leading period) associated with the
// detected type.
func DetectContentType(d []byte) (string, string) {
	mime := mimetype.Detect(d)
	return mime.String(), mime.Extension()
}

// BasicAuth returns the Authorization header value for HTTP Basic auth
func BasicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
