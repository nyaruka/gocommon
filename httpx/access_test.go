package httpx_test

import (
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/httpx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _, ipv4MappedIPv6Net, _ = net.ParseCIDR("::ffff:0:0/96")

func TestAccessConfig(t *testing.T) {
	defer httpx.SetRequestor(httpx.DefaultRequestor)

	access := httpx.NewAccessConfig(
		30*time.Second,
		[]net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		},
		[]*net.IPNet{
			{IP: net.IPv4(10, 0, 0, 0).To4(), Mask: net.CIDRMask(8, 32)},
			// IPv4 net with IP left in 16-byte form (net.IPv4 returns 16-byte) — must still
			// be treated as IPv4 so 192.168.x.x hosts are denied.
			{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},
			ipv4MappedIPv6Net,
		},
	)

	httpx.SetRequestor(httpx.NewMockRequestor(map[string][]*httpx.MockResponse{
		"https://nyaruka.com": {
			httpx.NewMockResponse(200, nil, nil),
		},
		"https://11.0.0.0": {
			httpx.NewMockResponse(200, nil, nil),
		},
		"https://8.8.8.8": {
			httpx.NewMockResponse(200, nil, nil),
		},
	}))

	tests := []struct {
		url     string
		allowed bool
	}{
		// allowed
		{"https://nyaruka.com", true},
		{"https://11.0.0.0", true},
		// a plain IPv4 host must not match an IPv6 disallowed net like ::ffff:0:0/96
		{"https://8.8.8.8", true},

		// denied by IP match
		{"https://localhost/path", false},
		{"https://LOCALHOST:80", false},
		{"http://foo.localtest.me", false},
		{"https://127.0.0.1", false},
		{"https://[::1]:80", false},
		{"https://[0:0:0:0:0:0:0:1]:80", false},
		{"https://[0000:0000:0000:0000:0000:0000:0000:0001]:80", false},

		// denied by network match
		{"https://10.1.0.0", false},
		{"https://10.0.1.0", false},
		{"https://10.0.0.1", false},
		// IPv4 net constructed with a 16-byte IP must still match IPv4 hosts
		{"https://192.168.1.1", false},
		// IPv4-mapped IPv6 must not bypass an IPv4 disallowed net
		{"https://[0:0:0:0:0:ffff:0a01:0000]:80", false}, // 10.1.0.0 mapped to IPv6
	}
	for _, tc := range tests {
		request, err := http.NewRequest("GET", tc.url, nil)
		require.NoError(t, err)

		_, err = httpx.DoTrace(http.DefaultClient, request, nil, access, -1)

		if tc.allowed {
			assert.NoError(t, err)
		} else {
			assert.Equal(t, err, httpx.ErrAccessConfig, "error message mismatch for url %s", tc.url)
		}
	}
}

// recordingBody is an io.ReadCloser which records whether it has been closed
type recordingBody struct {
	closed bool
}

func (b *recordingBody) Read(p []byte) (int, error) { return 0, io.EOF }
func (b *recordingBody) Close() error               { b.closed = true; return nil }

func TestAccessTransport(t *testing.T) {
	access := httpx.NewAccessConfig(
		30*time.Second,
		[]net.IP{net.ParseIP("127.0.0.1")},
		nil,
	)

	// allowed request delegates to the inner transport
	inner := httpx.NewMockRequestor(map[string][]*httpx.MockResponse{
		"https://8.8.8.8": {httpx.NewMockResponse(200, nil, nil)},
	})
	transport := httpx.WithAccess(inner, access)
	req, err := http.NewRequest("GET", "https://8.8.8.8", nil)
	require.NoError(t, err)
	resp, err := transport.RoundTrip(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Len(t, inner.Requests(), 1)

	// disallowed request returns ErrAccessConfig and never reaches the inner transport
	inner = httpx.NewMockRequestor(map[string][]*httpx.MockResponse{})
	transport = httpx.WithAccess(inner, access)
	req, err = http.NewRequest("GET", "https://127.0.0.1", nil)
	require.NoError(t, err)
	resp, err = transport.RoundTrip(req)
	assert.Equal(t, httpx.ErrAccessConfig, err)
	assert.Nil(t, resp)
	assert.Empty(t, inner.Requests())

	// a denied request with a non-nil body must have that body closed (http.RoundTripper contract)
	body := &recordingBody{}
	inner = httpx.NewMockRequestor(map[string][]*httpx.MockResponse{})
	transport = httpx.WithAccess(inner, access)
	req, err = http.NewRequest("POST", "https://127.0.0.1", body)
	require.NoError(t, err)
	resp, err = transport.RoundTrip(req)
	assert.Equal(t, httpx.ErrAccessConfig, err)
	assert.Nil(t, resp)
	assert.True(t, body.closed, "request body should be closed on the deny path")
	assert.Empty(t, inner.Requests())

	// an error from Allow (here a resolver timeout) is propagated as-is, not converted to ErrAccessConfig.
	// A 1ns resolve timeout forces a deterministic error without depending on real DNS resolution.
	timeoutAccess := httpx.NewAccessConfig(time.Nanosecond, []net.IP{net.ParseIP("127.0.0.1")}, nil)
	inner = httpx.NewMockRequestor(map[string][]*httpx.MockResponse{})
	transport = httpx.WithAccess(inner, timeoutAccess)
	req, err = http.NewRequest("GET", "https://example.com", nil)
	require.NoError(t, err)
	resp, err = transport.RoundTrip(req)
	assert.Error(t, err)
	assert.NotEqual(t, httpx.ErrAccessConfig, err)
	assert.Nil(t, resp)
	assert.Empty(t, inner.Requests())

	// a nil inner transport falls back to http.DefaultTransport and the result is usable (here the request
	// is denied before it would reach DefaultTransport, so no real network call is made)
	transport = httpx.WithAccess(nil, access)
	assert.NotNil(t, transport)
	req, err = http.NewRequest("GET", "https://127.0.0.1", nil)
	require.NoError(t, err)
	resp, err = transport.RoundTrip(req)
	assert.Equal(t, httpx.ErrAccessConfig, err)
	assert.Nil(t, resp)

	// a nil access config is a pass-through, even for an otherwise-denied host
	inner = httpx.NewMockRequestor(map[string][]*httpx.MockResponse{
		"https://127.0.0.1": {httpx.NewMockResponse(200, nil, nil)},
	})
	transport = httpx.WithAccess(inner, nil)
	req, err = http.NewRequest("GET", "https://127.0.0.1", nil)
	require.NoError(t, err)
	resp, err = transport.RoundTrip(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Len(t, inner.Requests(), 1)
}

func TestParseNetworkList(t *testing.T) {
	privateNetwork1 := &net.IPNet{IP: net.IPv4(10, 0, 0, 0).To4(), Mask: net.CIDRMask(8, 32)}
	privateNetwork2 := &net.IPNet{IP: net.IPv4(172, 16, 0, 0).To4(), Mask: net.CIDRMask(12, 32)}
	privateNetwork3 := &net.IPNet{IP: net.IPv4(192, 168, 0, 0).To4(), Mask: net.CIDRMask(16, 32)}

	linkLocalIPv4 := &net.IPNet{IP: net.IPv4(169, 254, 0, 0).To4(), Mask: net.CIDRMask(16, 32)}
	_, linkLocalIPv6, _ := net.ParseCIDR("fe80::/10")

	// test with mailroom defaults
	ips, ipNets, err := httpx.ParseNetworks(`127.0.0.1`, `::1`, `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `169.254.0.0/16`, `fe80::/10`)
	assert.NoError(t, err)
	assert.Equal(t, []net.IP{net.IPv4(127, 0, 0, 1), net.ParseIP(`::1`)}, ips)
	assert.Equal(t, []*net.IPNet{privateNetwork1, privateNetwork2, privateNetwork3, linkLocalIPv4, linkLocalIPv6}, ipNets)

	// test with empty
	ips, ipNets, err = httpx.ParseNetworks()
	assert.NoError(t, err)
	assert.Equal(t, []net.IP{}, ips)
	assert.Equal(t, []*net.IPNet{}, ipNets)

	// test with invalid IP
	_, _, err = httpx.ParseNetworks(`127.0.1`)
	assert.EqualError(t, err, `couldn't parse '127.0.1' as an IP address`)

	// test with invalid network
	_, _, err = httpx.ParseNetworks(`127.0.0.1/x`)
	assert.EqualError(t, err, `couldn't parse '127.0.0.1/x' as an IP network`)
}
