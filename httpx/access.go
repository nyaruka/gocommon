package httpx

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/context"
)

// AccessConfig configures what can be accessed
type AccessConfig struct {
	ResolveTimeout time.Duration
	DisallowedIPs  []net.IP
	DisallowedNets []*net.IPNet
}

// NewAccessConfig creates a new access config
func NewAccessConfig(resolveTimeout time.Duration, disallowedIPs []net.IP, disallowedNets []*net.IPNet) *AccessConfig {
	return &AccessConfig{
		ResolveTimeout: resolveTimeout,
		DisallowedIPs:  disallowedIPs,
		DisallowedNets: disallowedNets,
	}
}

// Allow determines whether the given request should be allowed
func (c *AccessConfig) Allow(request *http.Request) (bool, error) {
	host := strings.ToLower(request.URL.Hostname())

	ctx, cancel := context.WithTimeout(context.Background(), c.ResolveTimeout)
	defer cancel()

	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return false, err
	}

	// if any of the host's addresses appear in the disallowed list, deny the request
	for _, addr := range addrs {
		// Normalize IPv4-in-IPv6 to 4-byte form so an IPv4-mapped IPv6 address can't bypass
		// an IPv4 rule by being expressed as ::ffff:x.x.x.x.
		ip := addr.IP
		isV4 := ip.To4() != nil
		if isV4 {
			ip = ip.To4()
		}
		for _, disallowed := range c.DisallowedIPs {
			if ip.Equal(disallowed) {
				return false, nil
			}
		}
		for _, disallowed := range c.DisallowedNets {
			// Only check IPv4 hosts against IPv4 nets and IPv6 hosts against IPv6 nets.
			// Without this, an IPv6 net that projects into IPv4 space (e.g. ::ffff:0:0/96)
			// would match every IPv4 host, because IPNet.Contains strips the ::ffff: prefix
			// internally and compares as IPv4. Use mask size for family detection because
			// the net's IP can be stored in 16-byte form even for IPv4 (e.g. when built via
			// net.IPv4 without To4), but the mask reliably reflects the intended family.
			_, maskBits := disallowed.Mask.Size()
			netIsV4 := maskBits == 32
			if isV4 != netIsV4 {
				continue
			}
			if disallowed.Contains(ip) {
				return false, nil
			}
		}
	}
	return true, nil
}

// check applies the access config to the request, returning ErrAccessConfig if the request is denied, or the
// underlying error if the access check itself fails. A nil config permits everything.
func (c *AccessConfig) check(request *http.Request) error {
	if c == nil {
		return nil
	}
	allowed, err := c.Allow(request)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrAccessConfig
	}
	return nil
}

// accessTransport is an http.RoundTripper which enforces an AccessConfig before delegating to an inner transport.
type accessTransport struct {
	inner  http.RoundTripper
	access *AccessConfig
}

// NewAccessTransport creates an http.RoundTripper which enforces the given access config before delegating to the
// inner transport. A nil access config makes it a pass-through, so it's always safe to wrap. If inner is nil then
// http.DefaultTransport is used.
func NewAccessTransport(inner http.RoundTripper, access *AccessConfig) http.RoundTripper {
	if inner == nil {
		inner = http.DefaultTransport
	}
	return &accessTransport{inner: inner, access: access}
}

func (t *accessTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if err := t.access.check(request); err != nil {
		return nil, err
	}
	return t.inner.RoundTrip(request)
}

// ParseNetworks parses a list of IPs and IP networks (written in CIDR notation)
func ParseNetworks(addrs ...string) ([]net.IP, []*net.IPNet, error) {
	ips := make([]net.IP, 0, len(addrs))
	ipNets := make([]*net.IPNet, 0, len(addrs))

	for _, addr := range addrs {
		if strings.Contains(addr, "/") {
			_, ipNet, err := net.ParseCIDR(addr)
			if err != nil {
				return nil, nil, fmt.Errorf("couldn't parse '%s' as an IP network", addr)
			}
			ipNets = append(ipNets, ipNet)
		} else {
			ip := net.ParseIP(addr)
			if ip == nil {
				return nil, nil, fmt.Errorf("couldn't parse '%s' as an IP address", addr)
			}
			ips = append(ips, ip)
		}
	}

	return ips, ipNets, nil
}
