package event

import "encoding/json"

// Event struct
type Event struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
