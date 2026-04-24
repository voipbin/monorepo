package sock

import "encoding/json"

// Request struct
type Request struct {
	URI       string          `json:"uri"`
	Method    RequestMethod   `json:"method"`
	Publisher string          `json:"publisher"`
	DataType  string          `json:"data_type"`
	Data      json.RawMessage `json:"data,omitempty"`
	// RequestID is an optional request correlation ID propagated from
	// the inbound HTTP request through api-manager into downstream
	// RPC calls. See the design doc (docs/plans/2026-04-24-api-error-
	// response-codes-design.md §5.2) for the propagation chain.
	// Clients and managers receive it via logrus fields so inbound
	// and outbound log lines can be joined by the same ID.
	RequestID string `json:"request_id,omitempty"`
}

// Response struct
type Response struct {
	StatusCode int             `json:"status_code"`
	DataType   string          `json:"data_type"`
	Data       json.RawMessage `json:"data,omitempty"`
}

// Event struct
type Event struct {
	Type      string          `json:"type"`
	Publisher string          `json:"publisher"`
	DataType  string          `json:"data_type"`
	Data      json.RawMessage `json:"data,omitempty"`
}

// RequestMethod type
type RequestMethod string

// List of RequestMethod
const (
	RequestMethodPost   RequestMethod = "POST"
	RequestMethodGet    RequestMethod = "GET"
	RequestMethodPut    RequestMethod = "PUT"
	RequestMethodDelete RequestMethod = "DELETE"
)
