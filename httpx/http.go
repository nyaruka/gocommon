package httpx

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"

	"github.com/gabriel-vasile/mimetype"
)

// ErrResponseSize is returned when response size exceeds provided limit
var ErrResponseSize = errors.New("response body exceeds size limit")

// ErrAccessConfig is returned when provided access config prevents request
var ErrAccessConfig = errors.New("request not permitted by access config")

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
