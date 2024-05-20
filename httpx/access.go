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
		for _, disallowed := range c.DisallowedIPs {
			if addr.IP.Equal(disallowed) {
				return false, nil
			}
		}
		for _, disallowed := range c.DisallowedNets {
			if disallowed.Contains(addr.IP) {
				return false, nil
			}
		}
	}
	return true, nil
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
