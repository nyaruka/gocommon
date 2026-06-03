package httpx

import (
	"io"
	"net/http"
)

// bodyLimitTransport is an http.RoundTripper which bounds how many bytes can be read from each response body,
// delegating to an inner transport. Reading a body beyond the limit fails with ErrResponseSize.
type bodyLimitTransport struct {
	inner    http.RoundTripper
	maxBytes int
}

// WithBodyLimit wraps an http.RoundTripper so that reading more than maxBytes from any response body fails with
// ErrResponseSize; a body of exactly maxBytes is allowed. A value <= 0 disables the limit. If inner is nil then
// http.DefaultTransport is used.
//
// This bounds the bytes actually read from the network, so it's what guards against buffering an arbitrarily large
// response from an untrusted endpoint. To get that protection while also tracing, wrap this *inside* WithTracing so
// the limit applies before the body is buffered:
//
//	httpx.WithTracing(httpx.WithBodyLimit(inner, maxBytes))
//
// Wrapping the other way around (WithBodyLimit outside WithTracing) is ineffective, as WithTracing reads the full
// body into memory before the limit would be applied.
func WithBodyLimit(inner http.RoundTripper, maxBytes int) http.RoundTripper {
	if inner == nil {
		inner = http.DefaultTransport
	}
	return &bodyLimitTransport{inner: inner, maxBytes: maxBytes}
}

func (t *bodyLimitTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.inner.RoundTrip(request)
	if err != nil || t.maxBytes <= 0 || response == nil || response.Body == nil {
		return response, err
	}

	// wrap the body so reads beyond the limit surface ErrResponseSize to whoever consumes it
	response.Body = &limitedBody{inner: response.Body, left: int64(t.maxBytes)}
	return response, nil
}

var _ http.RoundTripper = (*bodyLimitTransport)(nil)

// limitedBody wraps a response body so that reading more than left bytes from it fails with ErrResponseSize. It
// permits reading one byte beyond the limit so that a body of exactly the limit is allowed while a larger one is
// reliably detected; the read that crosses the limit returns the within-limit bytes alongside ErrResponseSize and
// drops only the probe byte, following the io.Reader convention of reporting consumed bytes together with the error.
type limitedBody struct {
	inner io.ReadCloser
	left  int64
}

func (b *limitedBody) Read(p []byte) (int, error) {
	if b.left < 0 {
		return 0, ErrResponseSize
	}
	// read at most one byte beyond the remaining allowance so we can distinguish an exactly-at-limit body from one
	// that exceeds it
	if int64(len(p)) > b.left+1 {
		p = p[:b.left+1]
	}
	n, err := b.inner.Read(p)
	b.left -= int64(n)
	if b.left < 0 {
		// we read exactly one byte past the limit (n == old left + 1), so return the within-limit bytes and surface
		// the size error, dropping only that probe byte
		return n - 1, ErrResponseSize
	}
	return n, err
}

func (b *limitedBody) Close() error { return b.inner.Close() }
