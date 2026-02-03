package event

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEvent_JSONMarshal(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 123000000, time.UTC)
	e := &Event{
		Timestamp: ts,
		EventType: "activeflow_created",
		Publisher: "flow-manager",
		DataType:  "application/json",
		Data:      json.RawMessage(`{"key":"value"}`),
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled Event
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !unmarshaled.Timestamp.Equal(e.Timestamp) {
		t.Errorf("Timestamp = %v, want %v", unmarshaled.Timestamp, e.Timestamp)
	}

	if unmarshaled.EventType != e.EventType {
		t.Errorf("EventType = %q, want %q", unmarshaled.EventType, e.EventType)
	}

	if unmarshaled.Publisher != e.Publisher {
		t.Errorf("Publisher = %q, want %q", unmarshaled.Publisher, e.Publisher)
	}

	if unmarshaled.DataType != e.DataType {
		t.Errorf("DataType = %q, want %q", unmarshaled.DataType, e.DataType)
	}
}

func TestEvent_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"timestamp": "2024-01-15T10:30:00.123Z",
		"event_type": "activeflow_created",
		"publisher": "flow-manager",
		"data_type": "application/json",
		"data": {"key": "value"}
	}`

	var e Event
	if err := json.Unmarshal([]byte(jsonData), &e); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	expectedTime := time.Date(2024, 1, 15, 10, 30, 0, 123000000, time.UTC)
	if !e.Timestamp.Equal(expectedTime) {
		t.Errorf("Timestamp = %v, want %v", e.Timestamp, expectedTime)
	}

	if e.EventType != "activeflow_created" {
		t.Errorf("EventType = %q, want %q", e.EventType, "activeflow_created")
	}

	if e.Publisher != "flow-manager" {
		t.Errorf("Publisher = %q, want %q", e.Publisher, "flow-manager")
	}
}
