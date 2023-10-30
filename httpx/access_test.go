package httpx_test

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/nyaruka/gocommon/httpx"
	"github.com/stretchr/testify/assert"
)

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
		},
	)

	httpx.SetRequestor(httpx.NewMockRequestor(map[string][]*httpx.MockResponse{
		"https://nyaruka.com": {
			httpx.NewMockResponse(200, nil, nil),
		},
		"https://11.0.0.0": {
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
		{"https://[0:0:0:0:0:ffff:0a01:0000]:80", false}, // 10.1.0.0 mapped to IPv6
	}
	for _, tc := range tests {
		request, _ := http.NewRequest("GET", tc.url, nil)
		_, err := httpx.DoTrace(http.DefaultClient, request, nil, access, -1)

		if tc.allowed {
			assert.NoError(t, err)
		} else {
			assert.Equal(t, err, httpx.ErrAccessConfig, "error message mismatch for url %s", tc.url)
		}
	}
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
