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
				Name:            "Test Provider",
				Detail:          "Test provider detail",
				HealthStatus:    HealthStatusHealthy,
				HealthCheckedAt: timePtr(time.Date(2023, 1, 3, 12, 0, 0, 0, time.UTC)),
				TMCreate:        timePtr(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
				TMUpdate:        timePtr(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)),
				TMDelete:        nil,
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
				Name:            "Test Provider",
				Detail:          "Test provider detail",
				HealthStatus:    HealthStatusHealthy,
				HealthCheckedAt: timePtr(time.Date(2023, 1, 3, 12, 0, 0, 0, time.UTC)),
				TMCreate:        timePtr(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
				TMUpdate:        timePtr(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)),
				TMDelete:        nil,
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
		{
			name: "initial state - unknown health",
			provider: &Provider{
				ID:           uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				Type:         TypeSIP,
				Hostname:     "sip.new.com",
				HealthStatus: HealthStatusUnknown,
				// HealthCheckedAt: nil (zero value)
			},
			want: &WebhookMessage{
				ID:           uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				Type:         TypeSIP,
				Hostname:     "sip.new.com",
				HealthStatus: HealthStatusUnknown,
				// HealthCheckedAt: nil
			},
		},
		{
			name: "codecs populated",
			provider: &Provider{
				ID:       uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				Type:     TypeSIP,
				Hostname: "sip.codecs.com",
				Codecs:   "PCMU,PCMA",
			},
			want: &WebhookMessage{
				ID:       uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				Type:     TypeSIP,
				Hostname: "sip.codecs.com",
				Codecs:   "PCMU,PCMA",
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
			if got.Codecs != tt.want.Codecs {
				t.Errorf("Codecs = %v, want %v", got.Codecs, tt.want.Codecs)
			}
			if got.HealthStatus != tt.want.HealthStatus {
				t.Errorf("HealthStatus = %v, want %v", got.HealthStatus, tt.want.HealthStatus)
			}
			if (got.HealthCheckedAt == nil) != (tt.want.HealthCheckedAt == nil) {
				t.Errorf("HealthCheckedAt nil-ness = %v, want %v", got.HealthCheckedAt == nil, tt.want.HealthCheckedAt == nil)
			} else if got.HealthCheckedAt != nil && !got.HealthCheckedAt.Equal(*tt.want.HealthCheckedAt) {
				t.Errorf("HealthCheckedAt = %v, want %v", got.HealthCheckedAt, tt.want.HealthCheckedAt)
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

func Test_ConvertWebhookMessage_Codecs(t *testing.T) {
	p := &Provider{
		ID:     uuid.Must(uuid.NewV4()),
		Codecs: "PCMU,PCMA",
	}
	got := p.ConvertWebhookMessage()
	if got.Codecs != "PCMU,PCMA" {
		t.Errorf("expected Codecs %q, got %q", "PCMU,PCMA", got.Codecs)
	}
}
