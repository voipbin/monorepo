package provider

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func TestConvertWebhookMessage(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		want     *WebhookMessage
	}{
		{
			name: "all fields populated",
			provider: &Provider{
				ID:          uuid.FromStringOrNil("12345678-1234-1234-1234-123456789abc"),
				Type:        TypeSIP,
				Hostname:    "sip.example.com",
				TechPrefix:  "+1",
				TechPostfix: ";user=phone",
				TechHeaders: map[string]string{
					"X-Custom": "value",
				},
				Name:     "Test Provider",
				Detail:   "Test provider detail",
				TMCreate: timePtr(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
				TMUpdate: timePtr(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)),
				TMDelete: nil,
			},
			want: &WebhookMessage{
				ID:          uuid.FromStringOrNil("12345678-1234-1234-1234-123456789abc"),
				Type:        TypeSIP,
				Hostname:    "sip.example.com",
				TechPrefix:  "+1",
				TechPostfix: ";user=phone",
				TechHeaders: map[string]string{
					"X-Custom": "value",
				},
				Name:     "Test Provider",
				Detail:   "Test provider detail",
				TMCreate: timePtr(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
				TMUpdate: timePtr(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)),
				TMDelete: nil,
			},
		},
		{
			name: "minimal fields",
			provider: &Provider{
				ID:       uuid.FromStringOrNil("87654321-4321-4321-4321-cba987654321"),
				Type:     TypeSIP,
				Hostname: "sip.minimal.com",
			},
			want: &WebhookMessage{
				ID:       uuid.FromStringOrNil("87654321-4321-4321-4321-cba987654321"),
				Type:     TypeSIP,
				Hostname: "sip.minimal.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.provider.ConvertWebhookMessage()
			if got.ID != tt.want.ID {
				t.Errorf("ID = %v, want %v", got.ID, tt.want.ID)
			}
			if got.Type != tt.want.Type {
				t.Errorf("Type = %v, want %v", got.Type, tt.want.Type)
			}
			if got.Hostname != tt.want.Hostname {
				t.Errorf("Hostname = %v, want %v", got.Hostname, tt.want.Hostname)
			}
			if got.TechPrefix != tt.want.TechPrefix {
				t.Errorf("TechPrefix = %v, want %v", got.TechPrefix, tt.want.TechPrefix)
			}
			if got.TechPostfix != tt.want.TechPostfix {
				t.Errorf("TechPostfix = %v, want %v", got.TechPostfix, tt.want.TechPostfix)
			}
			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.Detail != tt.want.Detail {
				t.Errorf("Detail = %v, want %v", got.Detail, tt.want.Detail)
			}
		})
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		wantErr  bool
	}{
		{
			name: "valid provider",
			provider: &Provider{
				ID:          uuid.FromStringOrNil("12345678-1234-1234-1234-123456789abc"),
				Type:        TypeSIP,
				Hostname:    "sip.example.com",
				TechPrefix:  "+1",
				TechPostfix: ";user=phone",
				TechHeaders: map[string]string{
					"X-Custom": "value",
				},
				Name:   "Test Provider",
				Detail: "Test provider detail",
			},
			wantErr: false,
		},
		{
			name: "minimal provider",
			provider: &Provider{
				ID:       uuid.FromStringOrNil("87654321-4321-4321-4321-cba987654321"),
				Type:     TypeSIP,
				Hostname: "sip.minimal.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.provider.CreateWebhookEvent()
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateWebhookEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) == 0 {
					t.Errorf("CreateWebhookEvent() returned empty byte slice")
				}
				// Verify it's valid JSON
				var result WebhookMessage
				if err := json.Unmarshal(got, &result); err != nil {
					t.Errorf("CreateWebhookEvent() returned invalid JSON: %v", err)
				}
				// Verify ID matches
				if result.ID != tt.provider.ID {
					t.Errorf("CreateWebhookEvent() ID = %v, want %v", result.ID, tt.provider.ID)
				}
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
