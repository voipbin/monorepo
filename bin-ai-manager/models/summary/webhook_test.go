package summary

import (
	"encoding/json"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func TestConvertWebhookMessage(t *testing.T) {
	tests := []struct {
		name      string
		summary   *Summary
		wantNil   bool
		checkFunc func(t *testing.T, wh *WebhookMessage, s *Summary)
	}{
		{
			name: "converts_summary_with_all_fields",
			summary: &Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				ActiveflowID:  uuid.Must(uuid.NewV4()),
				OnEndFlowID:   uuid.Must(uuid.NewV4()),
				ReferenceType: ReferenceTypeCall,
				ReferenceID:   uuid.Must(uuid.NewV4()),
				Status:        StatusDone,
				Language:      "en",
				Content:       "Test summary content",
				TMCreate:      ptrTime(time.Now()),
				TMUpdate:      ptrTime(time.Now()),
			},
			checkFunc: func(t *testing.T, wh *WebhookMessage, s *Summary) {
				if wh.ID != s.ID {
					t.Errorf("Wrong ID. expect: %s, got: %s", s.ID, wh.ID)
				}
				if wh.CustomerID != s.CustomerID {
					t.Errorf("Wrong CustomerID. expect: %s, got: %s", s.CustomerID, wh.CustomerID)
				}
				if wh.ActiveflowID != s.ActiveflowID {
					t.Errorf("Wrong ActiveflowID. expect: %s, got: %s", s.ActiveflowID, wh.ActiveflowID)
				}
				if wh.OnEndFlowID != s.OnEndFlowID {
					t.Errorf("Wrong OnEndFlowID. expect: %s, got: %s", s.OnEndFlowID, wh.OnEndFlowID)
				}
				if wh.ReferenceType != s.ReferenceType {
					t.Errorf("Wrong ReferenceType. expect: %s, got: %s", s.ReferenceType, wh.ReferenceType)
				}
				if wh.ReferenceID != s.ReferenceID {
					t.Errorf("Wrong ReferenceID. expect: %s, got: %s", s.ReferenceID, wh.ReferenceID)
				}
				if wh.Status != s.Status {
					t.Errorf("Wrong Status. expect: %s, got: %s", s.Status, wh.Status)
				}
				if wh.Language != s.Language {
					t.Errorf("Wrong Language. expect: %s, got: %s", s.Language, wh.Language)
				}
				if wh.Content != s.Content {
					t.Errorf("Wrong Content. expect: %s, got: %s", s.Content, wh.Content)
				}
			},
		},
		{
			name: "converts_summary_with_empty_fields",
			summary: &Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.Nil,
					CustomerID: uuid.Nil,
				},
				ActiveflowID:  uuid.Nil,
				OnEndFlowID:   uuid.Nil,
				ReferenceType: ReferenceTypeNone,
				ReferenceID:   uuid.Nil,
				Status:        StatusNone,
			},
			checkFunc: func(t *testing.T, wh *WebhookMessage, s *Summary) {
				if wh.ID != s.ID {
					t.Errorf("Wrong ID. expect: %s, got: %s", s.ID, wh.ID)
				}
				if wh.Status != s.Status {
					t.Errorf("Wrong Status. expect: %s, got: %s", s.Status, wh.Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wh := tt.summary.ConvertWebhookMessage()
			if wh == nil && !tt.wantNil {
				t.Error("Expected non-nil webhook message, got nil")
				return
			}
			if wh != nil && tt.wantNil {
				t.Error("Expected nil webhook message, got non-nil")
				return
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, wh, tt.summary)
			}
		})
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	tests := []struct {
		name      string
		summary   *Summary
		wantError bool
	}{
		{
			name: "creates_webhook_event_successfully",
			summary: &Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				ActiveflowID:  uuid.Must(uuid.NewV4()),
				ReferenceType: ReferenceTypeConference,
				ReferenceID:   uuid.Must(uuid.NewV4()),
				Status:        StatusProgressing,
				Language:      "es",
				Content:       "Summary in progress",
				TMCreate:      ptrTime(time.Now()),
			},
			wantError: false,
		},
		{
			name: "creates_webhook_event_with_empty_summary",
			summary: &Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.Nil,
					CustomerID: uuid.Nil,
				},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.summary.CreateWebhookEvent()
			if (err != nil) != tt.wantError {
				t.Errorf("CreateWebhookEvent() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				// Verify it's valid JSON
				var wh WebhookMessage
				if errUnmarshal := json.Unmarshal(data, &wh); errUnmarshal != nil {
					t.Errorf("Failed to unmarshal webhook event: %v", errUnmarshal)
				}
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
