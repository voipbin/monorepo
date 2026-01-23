package webhook

import (
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
)

func TestWebhookStruct(t *testing.T) {
	customerID := uuid.Must(uuid.NewV4())

	w := Webhook{
		CustomerID: customerID,
		DataType:   DataTypeJSON,
		Data:       map[string]string{"key": "value"},
	}

	if w.CustomerID != customerID {
		t.Errorf("Webhook.CustomerID = %v, expected %v", w.CustomerID, customerID)
	}
	if w.DataType != DataTypeJSON {
		t.Errorf("Webhook.DataType = %v, expected %v", w.DataType, DataTypeJSON)
	}
	if w.Data == nil {
		t.Errorf("Webhook.Data should not be nil")
	}
}

func TestDataStruct(t *testing.T) {
	rawData := json.RawMessage(`{"message": "test"}`)

	d := Data{
		Type: "call_created",
		Data: rawData,
	}

	if d.Type != "call_created" {
		t.Errorf("Data.Type = %v, expected %v", d.Type, "call_created")
	}
	if d.Data == nil {
		t.Errorf("Data.Data should not be nil")
	}
}

func TestMethodTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant MethodType
		expected string
	}{
		{"method_post", MethodTypePOST, "POST"},
		{"method_put", MethodTypePUT, "PUT"},
		{"method_get", MethodTypeGET, "GET"},
		{"method_delete", MethodTypeDELETE, "DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestDataTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant DataType
		expected string
	}{
		{"data_type_empty", DataTypeEmpty, ""},
		{"data_type_json", DataTypeJSON, "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
