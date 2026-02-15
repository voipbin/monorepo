package tag

import (
	"encoding/json"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func TestConvertWebhookMessage(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name      string
		tag       *Tag
		expectRes *WebhookMessage
	}{
		{
			name: "converts_tag_to_webhook_message",
			tag: &Tag{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("250bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
					CustomerID: uuid.FromStringOrNil("350bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
				},
				Name:     "test name",
				Detail:   "test detail",
				TMCreate: curTime,
				TMUpdate: nil,
				TMDelete: nil,
			},
			expectRes: &WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("250bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
					CustomerID: uuid.FromStringOrNil("350bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
				},
				Name:     "test name",
				Detail:   "test detail",
				TMCreate: curTime,
				TMUpdate: nil,
				TMDelete: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.tag.ConvertWebhookMessage()

			if res.ID != tt.expectRes.ID {
				t.Errorf("Wrong ID. expect: %s, got: %s", tt.expectRes.ID, res.ID)
			}
			if res.CustomerID != tt.expectRes.CustomerID {
				t.Errorf("Wrong CustomerID. expect: %s, got: %s", tt.expectRes.CustomerID, res.CustomerID)
			}
			if res.Name != tt.expectRes.Name {
				t.Errorf("Wrong Name. expect: %s, got: %s", tt.expectRes.Name, res.Name)
			}
			if res.Detail != tt.expectRes.Detail {
				t.Errorf("Wrong Detail. expect: %s, got: %s", tt.expectRes.Detail, res.Detail)
			}
		})
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()

	tests := []struct {
		name    string
		tag     *Tag
		wantErr bool
	}{
		{
			name: "creates_webhook_event_successfully",
			tag: &Tag{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("250bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
					CustomerID: uuid.FromStringOrNil("350bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
				},
				Name:     "test name",
				Detail:   "test detail",
				TMCreate: curTime,
				TMUpdate: nil,
				TMDelete: nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.tag.CreateWebhookEvent()
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateWebhookEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// verify it's valid JSON
				var msg WebhookMessage
				if err := json.Unmarshal(res, &msg); err != nil {
					t.Errorf("Failed to unmarshal webhook event: %v", err)
				}

				if msg.ID != tt.tag.ID {
					t.Errorf("Wrong ID in webhook event. expect: %s, got: %s", tt.tag.ID, msg.ID)
				}
				if msg.Name != tt.tag.Name {
					t.Errorf("Wrong Name in webhook event. expect: %s, got: %s", tt.tag.Name, msg.Name)
				}
			}
		})
	}
}
