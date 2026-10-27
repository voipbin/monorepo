package mcpoauthstate

import (
	"testing"
	"time"
)

func Test_IsExpired(t *testing.T) {
	expire := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		expire *time.Time
		now    time.Time
		expect bool
	}{
		{name: "nil expiry fails closed", expire: nil, now: expire.Add(-time.Hour), expect: true},
		{name: "well before expiry", expire: &expire, now: expire.Add(-time.Hour), expect: false},
		{name: "one nanosecond before expiry", expire: &expire, now: expire.Add(-time.Nanosecond), expect: false},
		{name: "exactly at expiry", expire: &expire, now: expire, expect: true},
		{name: "one nanosecond after expiry", expire: &expire, now: expire.Add(time.Nanosecond), expect: true},
		{name: "well after expiry", expire: &expire, now: expire.Add(24 * time.Hour), expect: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &McpOAuthState{TMExpire: tt.expire}
			if got := s.IsExpired(tt.now); got != tt.expect {
				t.Errorf("IsExpired() = %v, expect: %v", got, tt.expect)
			}
		})
	}
}
