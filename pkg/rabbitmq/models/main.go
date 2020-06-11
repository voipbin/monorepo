package models

// Request struct
type Request struct {
	URI      string        `json:"uri"`
	Method   RequestMethod `json:"method"`
	DataType string        `json:"data_type"`
	Data     string        `json:"data"`
}

// Response struct
type Response struct {
	StatusCode int    `json:"status_code"`
	DataType   string `json:"data_type"`
	Data       string `json:"data"`
}

// Event struct
type Event struct {
	Type     EventType `json:"type"`
	DataType string    `json:"data_type"`
	Data     string    `json:"data"`
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

// EventType type
type EventType string

// List of EventType
const (
	EventTypeCall EventType = "cm_call"
)
