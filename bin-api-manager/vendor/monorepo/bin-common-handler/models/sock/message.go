package sock

import "encoding/json"

// Request struct
type Request struct {
	URI       string          `json:"uri"`
	Method    RequestMethod   `json:"method"`
	Publisher string          `json:"publisher"`
	DataType  string          `json:"data_type"`
	Data      json.RawMessage `json:"data,omitempty"`
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
