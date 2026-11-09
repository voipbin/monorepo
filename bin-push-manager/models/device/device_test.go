package device

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func Test_IsValidPlatform(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		expect   bool
	}{
		{"android", PlatformAndroid, true},
		{"ios is not supported yet", Platform("ios"), false},
		{"empty", Platform(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidPlatform(tt.platform); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}

func Test_IsLive(t *testing.T) {
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	at := func(d time.Duration) *time.Time { v := now.Add(-d); return &v }

	tests := []struct {
		name     string
		tmUpdate *time.Time
		expect   bool
	}{
		{"refreshed now", at(0), true},
		{"just inside the expiry", at(Expiry), true},
		{"just outside the expiry", at(Expiry + time.Nanosecond), false},
		{"never refreshed", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Device{TMUpdate: tt.tmUpdate}
			if got := d.IsLive(now); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}

func Test_tokenNeverInJSON(t *testing.T) {
	const token = "fake-token-value-1234567890"
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	d := &Device{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18"),
			CustomerID: uuid.FromStringOrNil("7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30"),
		},
		AgentID:  uuid.FromStringOrNil("b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52"),
		Platform: PlatformAndroid,
		Token:    token,
		TMCreate: &now,
		TMUpdate: &now,
	}

	for name, v := range map[string]any{"model": d, "webhook message": d.ConvertWebhookMessage()} {
		t.Run(name, func(t *testing.T) {
			m, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if strings.Contains(string(m), token) || strings.Contains(string(m), "token") {
				t.Errorf("Wrong match. the token must not be in the json. got: %s", m)
			}
		})
	}
}

func Test_ConvertWebhookMessage(t *testing.T) {
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	d := &Device{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18"),
			CustomerID: uuid.FromStringOrNil("7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30"),
		},
		AgentID:  uuid.FromStringOrNil("b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52"),
		Platform: PlatformAndroid,
		TMCreate: &now,
		TMUpdate: &now,
	}

	res := d.ConvertWebhookMessage()
	if res.ID != d.ID || res.CustomerID != d.CustomerID || res.AgentID != d.AgentID || res.Platform != d.Platform || res.TMCreate != d.TMCreate || res.TMUpdate != d.TMUpdate {
		t.Errorf("Wrong match. expect: %v, got: %v", d, res)
	}
}
