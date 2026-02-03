package event

import (
	"encoding/json"
	"time"
)

// Event represents a single event from ClickHouse.
type Event struct {
	Timestamp time.Time       `json:"timestamp"`
	EventType string          `json:"event_type"`
	Publisher string          `json:"publisher"`
	DataType  string          `json:"data_type"`
	Data      json.RawMessage `json:"data"`
}
