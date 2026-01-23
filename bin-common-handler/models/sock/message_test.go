package sock

import (
	"encoding/json"
	"testing"
)

func TestRequestStruct(t *testing.T) {
	data := json.RawMessage(`{"key": "value"}`)

	r := Request{
		URI:       "/v1/calls/123",
		Method:    RequestMethodGet,
		Publisher: "test-service",
		DataType:  "application/json",
		Data:      data,
	}

	if r.URI != "/v1/calls/123" {
		t.Errorf("Request.URI = %v, expected %v", r.URI, "/v1/calls/123")
	}
	if r.Method != RequestMethodGet {
		t.Errorf("Request.Method = %v, expected %v", r.Method, RequestMethodGet)
	}
	if r.Publisher != "test-service" {
		t.Errorf("Request.Publisher = %v, expected %v", r.Publisher, "test-service")
	}
	if r.DataType != "application/json" {
		t.Errorf("Request.DataType = %v, expected %v", r.DataType, "application/json")
	}
	if string(r.Data) != `{"key": "value"}` {
		t.Errorf("Request.Data = %v, expected %v", string(r.Data), `{"key": "value"}`)
	}
}

func TestResponseStruct(t *testing.T) {
	data := json.RawMessage(`{"result": "success"}`)

	r := Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	if r.StatusCode != 200 {
		t.Errorf("Response.StatusCode = %v, expected %v", r.StatusCode, 200)
	}
	if r.DataType != "application/json" {
		t.Errorf("Response.DataType = %v, expected %v", r.DataType, "application/json")
	}
	if string(r.Data) != `{"result": "success"}` {
		t.Errorf("Response.Data = %v, expected %v", string(r.Data), `{"result": "success"}`)
	}
}

func TestEventStruct(t *testing.T) {
	data := json.RawMessage(`{"event_data": "test"}`)

	e := Event{
		Type:      "call_created",
		Publisher: "call-manager",
		DataType:  "application/json",
		Data:      data,
	}

	if e.Type != "call_created" {
		t.Errorf("Event.Type = %v, expected %v", e.Type, "call_created")
	}
	if e.Publisher != "call-manager" {
		t.Errorf("Event.Publisher = %v, expected %v", e.Publisher, "call-manager")
	}
	if e.DataType != "application/json" {
		t.Errorf("Event.DataType = %v, expected %v", e.DataType, "application/json")
	}
	if string(e.Data) != `{"event_data": "test"}` {
		t.Errorf("Event.Data = %v, expected %v", string(e.Data), `{"event_data": "test"}`)
	}
}

func TestRequestMethodConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant RequestMethod
		expected string
	}{
		{"request_method_post", RequestMethodPost, "POST"},
		{"request_method_get", RequestMethodGet, "GET"},
		{"request_method_put", RequestMethodPut, "PUT"},
		{"request_method_delete", RequestMethodDelete, "DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestRequestWithDifferentMethods(t *testing.T) {
	tests := []struct {
		name   string
		method RequestMethod
		uri    string
	}{
		{"post_request", RequestMethodPost, "/v1/calls"},
		{"get_request", RequestMethodGet, "/v1/calls/123"},
		{"put_request", RequestMethodPut, "/v1/calls/123"},
		{"delete_request", RequestMethodDelete, "/v1/calls/123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Request{
				URI:    tt.uri,
				Method: tt.method,
			}
			if r.Method != tt.method {
				t.Errorf("Request.Method = %v, expected %v", r.Method, tt.method)
			}
			if r.URI != tt.uri {
				t.Errorf("Request.URI = %v, expected %v", r.URI, tt.uri)
			}
		})
	}
}

func TestResponseStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"ok", 200},
		{"created", 201},
		{"bad_request", 400},
		{"not_found", 404},
		{"internal_server_error", 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Response{
				StatusCode: tt.statusCode,
			}
			if r.StatusCode != tt.statusCode {
				t.Errorf("Response.StatusCode = %v, expected %v", r.StatusCode, tt.statusCode)
			}
		})
	}
}
