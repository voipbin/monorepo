package webhook

import "encoding/json"

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
