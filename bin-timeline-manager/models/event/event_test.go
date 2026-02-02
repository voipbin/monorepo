package event

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func TestEvent_JSONMarshal(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 123000000, time.UTC)
	e := &Event{
		Timestamp: ts,
		EventType: "activeflow_created",
		Publisher: "flow-manager",
		DataType:  "application/json",
		Data:      json.RawMessage(`{"key":"value"}`),
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled Event
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if !unmarshaled.Timestamp.Equal(e.Timestamp) {
		t.Errorf("Timestamp = %v, want %v", unmarshaled.Timestamp, e.Timestamp)
	}

	if unmarshaled.EventType != e.EventType {
		t.Errorf("EventType = %q, want %q", unmarshaled.EventType, e.EventType)
	}

	if unmarshaled.Publisher != e.Publisher {
		t.Errorf("Publisher = %q, want %q", unmarshaled.Publisher, e.Publisher)
	}

	if unmarshaled.DataType != e.DataType {
		t.Errorf("DataType = %q, want %q", unmarshaled.DataType, e.DataType)
	}
}

func TestEvent_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"timestamp": "2024-01-15T10:30:00.123Z",
		"event_type": "activeflow_created",
		"publisher": "flow-manager",
		"data_type": "application/json",
		"data": {"key": "value"}
	}`

	var e Event
	if err := json.Unmarshal([]byte(jsonData), &e); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	expectedTime := time.Date(2024, 1, 15, 10, 30, 0, 123000000, time.UTC)
	if !e.Timestamp.Equal(expectedTime) {
		t.Errorf("Timestamp = %v, want %v", e.Timestamp, expectedTime)
	}

	if e.EventType != "activeflow_created" {
		t.Errorf("EventType = %q, want %q", e.EventType, "activeflow_created")
	}

	if e.Publisher != "flow-manager" {
		t.Errorf("Publisher = %q, want %q", e.Publisher, "flow-manager")
	}
}

func TestEventListResponse_JSONMarshal(t *testing.T) {
	ts1 := time.Date(2024, 1, 15, 10, 30, 0, 123000000, time.UTC)
	ts2 := time.Date(2024, 1, 15, 10, 29, 0, 123000000, time.UTC)
	resp := &EventListResponse{
		Result: []*Event{
			{Timestamp: ts1, EventType: "activeflow_created"},
			{Timestamp: ts2, EventType: "activeflow_started"},
		},
		NextPageToken: "2024-01-15T10:29:00.123Z",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled EventListResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(unmarshaled.Result) != 2 {
		t.Errorf("len(Result) = %d, want 2", len(unmarshaled.Result))
	}

	if unmarshaled.NextPageToken != resp.NextPageToken {
		t.Errorf("NextPageToken = %q, want %q", unmarshaled.NextPageToken, resp.NextPageToken)
	}
}

func TestEventListResponse_EmptyResult(t *testing.T) {
	resp := &EventListResponse{
		Result: []*Event{},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled EventListResponse
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(unmarshaled.Result) != 0 {
		t.Errorf("len(Result) = %d, want 0", len(unmarshaled.Result))
	}

	if unmarshaled.NextPageToken != "" {
		t.Errorf("NextPageToken = %q, want empty", unmarshaled.NextPageToken)
	}
}

func TestEventListResponse_OmitEmptyNextPageToken(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 123000000, time.UTC)
	resp := &EventListResponse{
		Result: []*Event{
			{Timestamp: ts, EventType: "activeflow_created"},
		},
		NextPageToken: "",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Check that next_page_token is omitted when empty
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if _, exists := raw["next_page_token"]; exists {
		t.Error("next_page_token should be omitted when empty")
	}
}

func TestEventListRequest_JSONMarshal(t *testing.T) {
	testID := uuid.Must(uuid.NewV4())
	req := &EventListRequest{
		Publisher: "flow-manager",
		ID:        testID,
		Events:    []string{"activeflow_*", "flow_created"},
		PageToken: "2024-01-15T10:29:00.123Z",
		PageSize:  50,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled EventListRequest
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if unmarshaled.Publisher != req.Publisher {
		t.Errorf("Publisher = %q, want %q", unmarshaled.Publisher, req.Publisher)
	}

	if unmarshaled.ID != req.ID {
		t.Errorf("ID = %v, want %v", unmarshaled.ID, req.ID)
	}

	if len(unmarshaled.Events) != 2 {
		t.Errorf("len(Events) = %d, want 2", len(unmarshaled.Events))
	}

	if unmarshaled.PageToken != req.PageToken {
		t.Errorf("PageToken = %q, want %q", unmarshaled.PageToken, req.PageToken)
	}

	if unmarshaled.PageSize != req.PageSize {
		t.Errorf("PageSize = %d, want %d", unmarshaled.PageSize, req.PageSize)
	}
}

func TestEventListRequest_JSONUnmarshal(t *testing.T) {
	testID := uuid.Must(uuid.NewV4())
	jsonData := `{
		"publisher": "flow-manager",
		"id": "` + testID.String() + `",
		"events": ["activeflow_*"],
		"page_token": "2024-01-15T10:29:00.123Z",
		"page_size": 100
	}`

	var req EventListRequest
	if err := json.Unmarshal([]byte(jsonData), &req); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if req.Publisher != "flow-manager" {
		t.Errorf("Publisher = %q, want %q", req.Publisher, "flow-manager")
	}

	if req.ID != testID {
		t.Errorf("ID = %v, want %v", req.ID, testID)
	}

	if len(req.Events) != 1 {
		t.Errorf("len(Events) = %d, want 1", len(req.Events))
	}

	if req.PageSize != 100 {
		t.Errorf("PageSize = %d, want 100", req.PageSize)
	}
}

func TestEventListRequest_OmitEmptyFields(t *testing.T) {
	testID := uuid.Must(uuid.NewV4())
	req := &EventListRequest{
		Publisher: "flow-manager",
		ID:        testID,
		Events:    []string{"activeflow_*"},
		// PageToken and PageSize not set
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if _, exists := raw["page_token"]; exists {
		t.Error("page_token should be omitted when empty")
	}

	// page_size with value 0 should be omitted due to omitempty
	if val, exists := raw["page_size"]; exists && val.(float64) == 0 {
		t.Error("page_size should be omitted when 0")
	}
}

func TestConstants(t *testing.T) {
	if DefaultPageSize != 100 {
		t.Errorf("DefaultPageSize = %d, want 100", DefaultPageSize)
	}

	if MaxPageSize != 1000 {
		t.Errorf("MaxPageSize = %d, want 1000", MaxPageSize)
	}
}
