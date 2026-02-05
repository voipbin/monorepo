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
