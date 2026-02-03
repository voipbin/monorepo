package event

import (
	"encoding/json"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
)

// Event represents a single event from ClickHouse.
type Event struct {
	Timestamp time.Time                 `json:"timestamp"`
	EventType string                    `json:"event_type"`
	Publisher commonoutline.ServiceName `json:"publisher"`
	DataType  string                    `json:"data_type"`
	Data      json.RawMessage           `json:"data"`
}
