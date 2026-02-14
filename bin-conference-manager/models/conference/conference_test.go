package conference

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func Test_Conference_MarshalUnmarshal(t *testing.T) {

	tmEnd := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	tmCreate := time.Date(2025, 4, 2, 0, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2025, 4, 3, 0, 0, 0, 0, time.UTC)
	tmDelete := time.Date(2025, 4, 4, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		conference Conference
	}{
		{
			name: "normal",
			conference: Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72124710-1e19-11f0-b975-6310cee17b54"),
					CustomerID: uuid.FromStringOrNil("7260700c-1e19-11f0-81f6-d71f167f930d"),
				},
				ConfbridgeID: uuid.FromStringOrNil("6f6934f0-1350-11ed-8084-2f82e5efd9c2"),
				Type:         TypeConference,
				Status:       StatusProgressing,
				Name:         "Test Conference",
				Detail:       "This is a test conference",
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
			// Marshal the conference to JSON
			data, err := json.Marshal(tt.conference)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// Unmarshal the JSON back to a Conference object
			var unmarshaled Conference
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// Compare the original and unmarshaled Conference objects
			if !reflect.DeepEqual(tt.conference, unmarshaled) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.conference, unmarshaled)
			}
		})
	}
}

func TestMatches(t *testing.T) {
	tmCreate := time.Date(2025, 4, 2, 0, 0, 0, 0, time.UTC)
	tmCreate2 := time.Date(2025, 4, 3, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		a        *Conference
		b        *Conference
		expected bool
	}{
		{
			name: "identical conferences",
			a: &Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72124710-1e19-11f0-b975-6310cee17b54"),
					CustomerID: uuid.FromStringOrNil("7260700c-1e19-11f0-81f6-d71f167f930d"),
				},
				Name:     "Test",
				TMCreate: &tmCreate,
			},
			b: &Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72124710-1e19-11f0-b975-6310cee17b54"),
					CustomerID: uuid.FromStringOrNil("7260700c-1e19-11f0-81f6-d71f167f930d"),
				},
				Name:     "Test",
				TMCreate: &tmCreate,
			},
			expected: true,
		},
		{
			name: "different create times should match",
			a: &Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72124710-1e19-11f0-b975-6310cee17b54"),
					CustomerID: uuid.FromStringOrNil("7260700c-1e19-11f0-81f6-d71f167f930d"),
				},
				Name:     "Test",
				TMCreate: &tmCreate,
			},
			b: &Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72124710-1e19-11f0-b975-6310cee17b54"),
					CustomerID: uuid.FromStringOrNil("7260700c-1e19-11f0-81f6-d71f167f930d"),
				},
				Name:     "Test",
				TMCreate: &tmCreate2,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.a.Matches(tt.b)
			if result != tt.expected {
				t.Errorf("Matches() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestString(t *testing.T) {
	conf := &Conference{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("72124710-1e19-11f0-b975-6310cee17b54"),
			CustomerID: uuid.FromStringOrNil("7260700c-1e19-11f0-81f6-d71f167f930d"),
		},
		Name: "Test Conference",
	}

	str := conf.String()
	if str == "" {
		t.Error("String() returned empty string")
	}
}

func TestConvertWebhookMessage(t *testing.T) {
	tmEnd := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	tmCreate := time.Date(2025, 4, 2, 0, 0, 0, 0, time.UTC)

	conf := &Conference{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("72124710-1e19-11f0-b975-6310cee17b54"),
			CustomerID: uuid.FromStringOrNil("7260700c-1e19-11f0-81f6-d71f167f930d"),
		},
		Type:   TypeConference,
		Status: StatusProgressing,
		Name:   "Test Conference",
		Detail: "Test Detail",
		Data: map[string]any{
			"key": "value",
		},
		Timeout:    300,
		PreFlowID:  uuid.FromStringOrNil("14e81da6-0e44-11f0-aee7-a3902cfbf3fc"),
		PostFlowID: uuid.FromStringOrNil("6fabe796-1350-11ed-a9be-63d034c16c8d"),
		ConferencecallIDs: []uuid.UUID{
			uuid.FromStringOrNil("6fdaccaa-1350-11ed-8a93-cb0e3c8d6bf8"),
		},
		RecordingID: uuid.FromStringOrNil("d0278cd8-1350-11ed-91f4-4f03d0c82169"),
		RecordingIDs: []uuid.UUID{
			uuid.FromStringOrNil("d0278cd8-1350-11ed-91f4-4f03d0c82169"),
		},
		TranscribeID: uuid.FromStringOrNil("e0278cd8-1350-11ed-91f4-4f03d0c82169"),
		TranscribeIDs: []uuid.UUID{
			uuid.FromStringOrNil("e0278cd8-1350-11ed-91f4-4f03d0c82169"),
		},
		TMEnd:    &tmEnd,
		TMCreate: &tmCreate,
	}

	webhook := conf.ConvertWebhookMessage()

	if webhook.ID != conf.ID {
		t.Errorf("WebhookMessage.ID = %v, expected %v", webhook.ID, conf.ID)
	}
	if webhook.Type != conf.Type {
		t.Errorf("WebhookMessage.Type = %v, expected %v", webhook.Type, conf.Type)
	}
	if webhook.Status != conf.Status {
		t.Errorf("WebhookMessage.Status = %v, expected %v", webhook.Status, conf.Status)
	}
	if webhook.Name != conf.Name {
		t.Errorf("WebhookMessage.Name = %v, expected %v", webhook.Name, conf.Name)
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	conf := &Conference{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("72124710-1e19-11f0-b975-6310cee17b54"),
			CustomerID: uuid.FromStringOrNil("7260700c-1e19-11f0-81f6-d71f167f930d"),
		},
		Type:   TypeConference,
		Status: StatusProgressing,
		Name:   "Test Conference",
	}

	data, err := conf.CreateWebhookEvent()
	if err != nil {
		t.Errorf("CreateWebhookEvent failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("CreateWebhookEvent returned empty data")
	}
}

func TestConvertStringMapToFieldMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		wantErr  bool
		validate func(t *testing.T, result map[Field]any)
	}{
		{
			name: "convert type field",
			input: map[string]string{
				"type": "conference",
			},
			wantErr: false,
			validate: func(t *testing.T, result map[Field]any) {
				if len(result) == 0 {
					t.Error("Expected non-empty result")
				}
			},
		},
		{
			name: "convert status field",
			input: map[string]string{
				"status": "progressing",
			},
			wantErr: false,
			validate: func(t *testing.T, result map[Field]any) {
				if len(result) == 0 {
					t.Error("Expected non-empty result")
				}
			},
		},
		{
			name:    "empty input",
			input:   map[string]string{},
			wantErr: false,
			validate: func(t *testing.T, result map[Field]any) {
				if len(result) != 0 {
					t.Errorf("Expected empty result, got %d fields", len(result))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertStringMapToFieldMap(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertStringMapToFieldMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}
