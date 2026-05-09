package providerhandler

import (
	"fmt"
	"strings"
)

func validateCodecs(s string) (string, error) {
	if len(s) > 255 {
		return "", fmt.Errorf("codecs string exceeds 255 characters")
	}
	if strings.ContainsAny(s, "\r\n") {
		return "", fmt.Errorf("codecs must not contain CRLF characters")
	}
	if strings.ContainsAny(s, "()") {
		return "", fmt.Errorf("codecs must not contain parentheses")
	}
	if s == "" {
		return "", nil
	}
	parts := strings.Split(s, ",")
	normalized := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return "", fmt.Errorf("codecs must not contain empty list elements (double comma)")
		}
		normalized = append(normalized, p)
	}
	return strings.Join(normalized, ","), nil
}
