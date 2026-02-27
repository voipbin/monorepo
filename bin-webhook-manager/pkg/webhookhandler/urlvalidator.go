package webhookhandler

import (
	"fmt"
	"net"
	"net/url"
)

// privateNetworks defines IP ranges that webhook URLs must not resolve to.
var privateNetworks = []net.IPNet{
	// IPv4 private/reserved
	{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
	{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},
	{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},
	{IP: net.IPv4(127, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
	{IP: net.IPv4(169, 254, 0, 0), Mask: net.CIDRMask(16, 32)}, // link-local / cloud metadata
	{IP: net.IPv4(0, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
	{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}, // CGN

	// IPv6 private/reserved
	{IP: net.ParseIP("::1"), Mask: net.CIDRMask(128, 128)},
	{IP: net.ParseIP("fc00::"), Mask: net.CIDRMask(7, 128)},
	{IP: net.ParseIP("fe80::"), Mask: net.CIDRMask(10, 128)},
}

// validateWebhookURL checks that a URL is safe for outbound webhook delivery.
func validateWebhookURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("empty URL")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("empty hostname")
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("cannot resolve hostname %s: %w", host, err)
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return fmt.Errorf("webhook URL resolves to private/reserved IP: %s -> %s", host, ip)
		}
	}

	return nil
}

func isPrivateIP(ip net.IP) bool {
	for _, network := range privateNetworks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
