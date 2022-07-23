package webhook

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// Webhook struct
type Webhook struct {
	CustomerID uuid.UUID   `json:"customer_id"`
	DataType   DataType    `json:"data_type"` // application/json
	Data       interface{} `json:"data"`
}

// Data defines common webhook data type.
type Data struct {
	Type string          `json:"type"` // message type
	Data json.RawMessage `json:"data"` // data
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
