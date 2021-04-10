package webhook

import "encoding/json"

// Webhook struct
type Webhook struct {
	Method     MethodType      `json:"method"`
	WebhookURI string          `json:"webhook_uri"`
	DataType   DataType        `json:"data_type"`
	Data       json.RawMessage `json:"data"`
}

// MethodType defines http method
type MethodType string

// list of Method
const (
	MethodTypePOST   MethodType = "POST"
	MethodTypePUT    MethodType = "PUT"
	MethodTypeGET    MethodType = "GET"
	MethodTypeDELETE MethodType = "DELETE"
)

// DataType defines the send data
type DataType string

// list of DataType
const (
	DataTypeEmpty DataType = ""
	DataTypeJSON  DataType = "application/json"
)
