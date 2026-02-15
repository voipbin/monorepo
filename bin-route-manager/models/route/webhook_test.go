package route

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func TestConvertWebhookMessage(t *testing.T) {
	tests := []struct {
		name  string
		route *Route
		want  *WebhookMessage
	}{
		{
			name: "all fields populated",
			route: &Route{
				ID:         uuid.FromStringOrNil("12345678-1234-1234-1234-123456789abc"),
				CustomerID: uuid.FromStringOrNil("87654321-4321-4321-4321-cba987654321"),
				Name:       "Test Route",
				Detail:     "Test route detail",
				ProviderID: uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555"),
				Priority:   1,
				Target:     TargetAll,
				TMCreate:   timePtr(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
				TMUpdate:   timePtr(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)),
				TMDelete:   nil,
			},
			want: &WebhookMessage{
				ID:         uuid.FromStringOrNil("12345678-1234-1234-1234-123456789abc"),
				CustomerID: uuid.FromStringOrNil("87654321-4321-4321-4321-cba987654321"),
				Name:       "Test Route",
				Detail:     "Test route detail",
				ProviderID: uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555"),
				Priority:   1,
				Target:     TargetAll,
				TMCreate:   timePtr(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
				TMUpdate:   timePtr(time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)),
				TMDelete:   nil,
			},
		},
		{
			name: "minimal fields",
			route: &Route{
				ID:         uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
				CustomerID: uuid.FromStringOrNil("ffffffff-eeee-dddd-cccc-bbbbbbbbbbbb"),
				ProviderID: uuid.FromStringOrNil("99999999-8888-7777-6666-555555555555"),
			},
			want: &WebhookMessage{
				ID:         uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
				CustomerID: uuid.FromStringOrNil("ffffffff-eeee-dddd-cccc-bbbbbbbbbbbb"),
				ProviderID: uuid.FromStringOrNil("99999999-8888-7777-6666-555555555555"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.route.ConvertWebhookMessage()
			if got.ID != tt.want.ID {
				t.Errorf("ID = %v, want %v", got.ID, tt.want.ID)
			}
			if got.CustomerID != tt.want.CustomerID {
				t.Errorf("CustomerID = %v, want %v", got.CustomerID, tt.want.CustomerID)
			}
			if got.ProviderID != tt.want.ProviderID {
				t.Errorf("ProviderID = %v, want %v", got.ProviderID, tt.want.ProviderID)
			}
			if got.Name != tt.want.Name {
				t.Errorf("Name = %v, want %v", got.Name, tt.want.Name)
			}
			if got.Detail != tt.want.Detail {
				t.Errorf("Detail = %v, want %v", got.Detail, tt.want.Detail)
			}
			if got.Priority != tt.want.Priority {
				t.Errorf("Priority = %v, want %v", got.Priority, tt.want.Priority)
			}
			if got.Target != tt.want.Target {
				t.Errorf("Target = %v, want %v", got.Target, tt.want.Target)
			}
		})
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	tests := []struct {
		name    string
		route   *Route
		wantErr bool
	}{
		{
			name: "valid route",
			route: &Route{
				ID:         uuid.FromStringOrNil("12345678-1234-1234-1234-123456789abc"),
				CustomerID: uuid.FromStringOrNil("87654321-4321-4321-4321-cba987654321"),
				Name:       "Test Route",
				Detail:     "Test route detail",
				ProviderID: uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555"),
				Priority:   1,
				Target:     TargetAll,
			},
			wantErr: false,
		},
		{
			name: "minimal route",
			route: &Route{
				ID:         uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
				CustomerID: uuid.FromStringOrNil("ffffffff-eeee-dddd-cccc-bbbbbbbbbbbb"),
				ProviderID: uuid.FromStringOrNil("99999999-8888-7777-6666-555555555555"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.route.CreateWebhookEvent()
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
				if result.ID != tt.route.ID {
					t.Errorf("CreateWebhookEvent() ID = %v, want %v", result.ID, tt.route.ID)
				}
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
