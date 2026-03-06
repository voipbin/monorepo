package ngclient

import (
	"testing"
	"time"
)

func TestNewNGClient(t *testing.T) {
	c, err := New("127.0.0.1:22222", 5*time.Second)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer c.Close()
}

func TestCookieUniqueness(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		c := newCookie()
		if seen[c] {
			t.Errorf("duplicate cookie: %q", c)
		}
		seen[c] = true
	}
}
