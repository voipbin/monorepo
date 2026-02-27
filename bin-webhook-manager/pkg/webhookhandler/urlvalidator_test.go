package webhookhandler

import (
	"net"
	"testing"
)

func Test_validateWebhookURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		// Valid URLs
		{"valid https", "https://example.com/webhook", false},
		{"valid http", "http://example.com/webhook", false},
		{"valid with port", "https://example.com:8080/webhook", false},

		// Invalid schemes
		{"no scheme", "example.com/webhook", true},
		{"file scheme", "file:///etc/passwd", true},
		{"ftp scheme", "ftp://example.com/file", true},
		{"empty string", "", true},

		// Private/reserved IPs
		{"localhost", "http://localhost/webhook", true},
		{"127.0.0.1", "http://127.0.0.1/webhook", true},
		{"10.x.x.x", "http://10.0.0.1/webhook", true},
		{"172.16.x.x", "http://172.16.0.1/webhook", true},
		{"192.168.x.x", "http://192.168.1.1/webhook", true},
		{"metadata endpoint", "http://169.254.169.254/computeMetadata/v1/", true},
		{"0.0.0.0", "http://0.0.0.0/webhook", true},
		{"cgn range", "http://100.64.0.1/webhook", true},
		{"ipv6 loopback", "http://[::1]/webhook", true},

		// Edge cases
		{"no host", "http:///path", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWebhookURL(tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateWebhookURL(%q) error = %v, wantErr %v", tt.rawURL, err, tt.wantErr)
			}
		})
	}
}

func Test_isPrivateIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		private bool
	}{
		{"loopback", "127.0.0.1", true},
		{"private 10", "10.0.0.1", true},
		{"private 172", "172.16.0.1", true},
		{"private 192", "192.168.1.1", true},
		{"metadata", "169.254.169.254", true},
		{"cgn", "100.64.0.1", true},
		{"public", "8.8.8.8", false},
		{"public 2", "93.184.216.34", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if isPrivateIP(ip) != tt.private {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, !tt.private, tt.private)
			}
		})
	}
}
