package event

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestEventStruct(t *testing.T) {
	rawData := json.RawMessage(`{"id": "123", "status": "active"}`)

	e := Event{
		Type: "call_created",
		Data: rawData,
	}

	if e.Type != "call_created" {
		t.Errorf("Event.Type = %v, expected %v", e.Type, "call_created")
	}
	if e.Data == nil {
		t.Errorf("Event.Data should not be nil")
	}
}

func TestEventMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name  string
		event Event
	}{
		{
			name: "simple_event",
			event: Event{
				Type: "message_created",
				Data: json.RawMessage(`{"text":"Hello"}`),
			},
		},
		{
			name: "complex_event",
			event: Event{
				Type: "call_hangup",
				Data: json.RawMessage(`{"call_id":"abc-123","duration":120,"hangup_cause":"normal"}`),
			},
		},
		{
			name: "empty_data_event",
			event: Event{
				Type: "system_status",
				Data: json.RawMessage(`{}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.event)
			if err != nil {
				t.Errorf("Failed to marshal event: %v", err)
				return
			}

			var result Event
			if err := json.Unmarshal(data, &result); err != nil {
				t.Errorf("Failed to unmarshal event: %v", err)
				return
			}

			if result.Type != tt.event.Type {
				t.Errorf("Event.Type = %v, expected %v", result.Type, tt.event.Type)
			}

			if !reflect.DeepEqual([]byte(result.Data), []byte(tt.event.Data)) {
				t.Errorf("Event.Data mismatch.\nexpect: %s\ngot: %s", tt.event.Data, result.Data)
			}
		})
	}
}

func TestEventEmptyType(t *testing.T) {
	e := Event{
		Type: "",
		Data: json.RawMessage(`{"data": "test"}`),
	}

	if e.Type != "" {
		t.Errorf("Event.Type should be empty, got %v", e.Type)
	}
}
