package conference

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func Test_Conference_CreateWebhookEvent(t *testing.T) {

	tmEnd := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	tmCreate := time.Date(2025, 4, 2, 0, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2025, 4, 3, 0, 0, 0, 0, time.UTC)
	tmDelete := time.Date(2025, 4, 4, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		conference Conference
		wantErr    bool
	}{
		{
			name: "normal",
			conference: Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72124710-1e19-11f0-b975-6310cee17b54"),
					CustomerID: uuid.FromStringOrNil("7260700c-1e19-11f0-81f6-d71f167f930d"),
				},
				Type:   TypeConference,
				Status: StatusProgressing,
				Name:   "Test Conference",
				Detail: "This is a test conference",
				Data: map[string]any{
					"key1": "value1",
					"key2": float64(2),
				},
				Timeout:    300,
				PreFlowID:  uuid.FromStringOrNil("14e81da6-0e44-11f0-aee7-a3902cfbf3fc"),
				PostFlowID: uuid.FromStringOrNil("6fabe796-1350-11ed-a9be-63d034c16c8d"),
				ConferencecallIDs: []uuid.UUID{
					uuid.FromStringOrNil("6fdaccaa-1350-11ed-8a93-cb0e3c8d6bf8"),
					uuid.FromStringOrNil("d0278cd8-1350-11ed-91f4-4f03d0c82169"),
				},
				RecordingID: uuid.FromStringOrNil("d0278cd8-1350-11ed-91f4-4f03d0c82169"),
				RecordingIDs: []uuid.UUID{
					uuid.FromStringOrNil("d0278cd8-1350-11ed-91f4-4f03d0c82169"),
				},
				TranscribeID: uuid.FromStringOrNil("d0278cd8-1350-11ed-91f4-4f03d0c82169"),
				TranscribeIDs: []uuid.UUID{
					uuid.FromStringOrNil("d0278cd8-1350-11ed-91f4-4f03d0c82169"),
				},
				TMEnd:    &tmEnd,
				TMCreate: &tmCreate,
				TMUpdate: &tmUpdate,
				TMDelete: &tmDelete,
			},
			wantErr: false,
		},
		{
			name: "empty conference",
			conference: Conference{
				Identity: commonidentity.Identity{},
			},
			wantErr: false,
		},
		{
			name: "invalid UUIDs",
			conference: Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.Nil,
					CustomerID: uuid.Nil,
				},
				PreFlowID:  uuid.Nil,
				PostFlowID: uuid.Nil,
			},
			wantErr: false,
		},
		{
			name: "missing optional fields",
			conference: Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72124710-1e19-11f0-b975-6310cee17b54"),
					CustomerID: uuid.FromStringOrNil("7260700c-1e19-11f0-81f6-d71f167f930d"),
				},
				Type:   TypeConference,
				Status: StatusProgressing,
			},
			wantErr: false,
		},
		{
			name: "invalid JSON data",
			conference: Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72124710-1e19-11f0-b975-6310cee17b54"),
					CustomerID: uuid.FromStringOrNil("7260700c-1e19-11f0-81f6-d71f167f930d"),
				},
				Data: map[string]any{
					"key1": func() {}, // Invalid type for JSON serialization
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.conference.CreateWebhookEvent()
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateWebhookEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_Webhook_MarshalUnmarshal(t *testing.T) {

	tmEnd := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	tmCreate := time.Date(2025, 4, 2, 0, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2025, 4, 3, 0, 0, 0, 0, time.UTC)
	tmDelete := time.Date(2025, 4, 4, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		webhook WebhookMessage
	}{
		{
			name: "normal",
			webhook: WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72124710-1e19-11f0-b975-6310cee17b54"),
					CustomerID: uuid.FromStringOrNil("7260700c-1e19-11f0-81f6-d71f167f930d"),
				},
				Type:   TypeConference,
				Status: StatusProgressing,
				Name:   "Test Conference",
				Detail: "This is a test conference",
				Data: map[string]any{
					"key1": "value1",
					"key2": float64(2),
				},
				Timeout:    300,
				PreFlowID:  uuid.FromStringOrNil("14e81da6-0e44-11f0-aee7-a3902cfbf3fc"),
				PostFlowID: uuid.FromStringOrNil("6fabe796-1350-11ed-a9be-63d034c16c8d"),
				ConferencecallIDs: []uuid.UUID{
					uuid.FromStringOrNil("6fdaccaa-1350-11ed-8a93-cb0e3c8d6bf8"),
					uuid.FromStringOrNil("d0278cd8-1350-11ed-91f4-4f03d0c82169"),
				},
				RecordingID: uuid.FromStringOrNil("d0278cd8-1350-11ed-91f4-4f03d0c82169"),
				RecordingIDs: []uuid.UUID{
					uuid.FromStringOrNil("d0278cd8-1350-11ed-91f4-4f03d0c82169"),
				},
				TranscribeID: uuid.FromStringOrNil("d0278cd8-1350-11ed-91f4-4f03d0c82169"),
				TranscribeIDs: []uuid.UUID{
					uuid.FromStringOrNil("d0278cd8-1350-11ed-91f4-4f03d0c82169"),
				},
				TMEnd:    &tmEnd,
				TMCreate: &tmCreate,
				TMUpdate: &tmUpdate,
				TMDelete: &tmDelete,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.webhook)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			var unmarshaled WebhookMessage
			err = json.Unmarshal(got, &unmarshaled)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// Compare the original and unmarshaled WebhookMessage objects
			if !reflect.DeepEqual(tt.webhook, unmarshaled) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.webhook, unmarshaled)
			}
		})
	}
}
