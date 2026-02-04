package dbhandler

import (
	"context"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
)

func TestBuildEventConditions(t *testing.T) {
	tests := []struct {
		name     string
		events   []string
		expected string
	}{
		{
			name:     "exact match",
			events:   []string{"activeflow_created"},
			expected: "event_type = 'activeflow_created'",
		},
		{
			name:     "wildcard prefix",
			events:   []string{"activeflow_*"},
			expected: "event_type LIKE 'activeflow_%'",
		},
		{
			name:     "multiple patterns",
			events:   []string{"activeflow_created", "flow_*"},
			expected: "event_type = 'activeflow_created' OR event_type LIKE 'flow_%'",
		},
		{
			name:     "wildcard all",
			events:   []string{"*"},
			expected: "",
		},
		{
			name:     "empty",
			events:   []string{},
			expected: "",
		},
		{
			name:     "single exact match",
			events:   []string{"call_hangup"},
			expected: "event_type = 'call_hangup'",
		},
		{
			name:     "multiple exact matches",
			events:   []string{"activeflow_created", "activeflow_started", "activeflow_finished"},
			expected: "event_type = 'activeflow_created' OR event_type = 'activeflow_started' OR event_type = 'activeflow_finished'",
		},
		{
			name:     "multiple wildcards",
			events:   []string{"activeflow_*", "flow_*", "call_*"},
			expected: "event_type LIKE 'activeflow_%' OR event_type LIKE 'flow_%' OR event_type LIKE 'call_%'",
		},
		{
			name:     "mixed patterns",
			events:   []string{"activeflow_created", "flow_*", "call_hangup"},
			expected: "event_type = 'activeflow_created' OR event_type LIKE 'flow_%' OR event_type = 'call_hangup'",
		},
		{
			name:     "wildcard all in middle",
			events:   []string{"activeflow_created", "*", "flow_created"},
			expected: "",
		},
		{
			name:     "empty string in events",
			events:   []string{""},
			expected: "event_type = ''",
		},
		{
			name:     "underscore wildcard only",
			events:   []string{"_*"},
			expected: "event_type LIKE '_%'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildEventConditions(tt.events)
			if result != tt.expected {
				t.Errorf("buildEventConditions(%v) = %q, want %q", tt.events, result, tt.expected)
			}
		})
	}
}

func TestEventList_NoConnection(t *testing.T) {
	handler := &dbHandler{
		address:  "localhost:9000",
		database: "test",
		conn:     nil, // No connection established
	}

	ctx := context.Background()
	testID := uuid.Must(uuid.NewV4())

	_, err := handler.EventList(ctx, "flow-manager", testID, []string{"activeflow_*"}, "", 10)
	if err == nil {
		t.Error("EventList() expected error when conn is nil, got nil")
	}

	if err.Error() != "clickhouse connection not established" {
		t.Errorf("EventList() error = %q, want %q", err.Error(), "clickhouse connection not established")
	}
}

func TestNewHandler(t *testing.T) {
	// Test that NewHandler returns a non-nil handler
	// Note: This doesn't actually connect since connectionLoop runs in goroutine
	handler := NewHandler("localhost:9000", "test")
	if handler == nil {
		t.Error("NewHandler() returned nil")
	}
}

func TestDBHandler_Interface(t *testing.T) {
	// Ensure dbHandler implements DBHandler interface
	var _ DBHandler = (*dbHandler)(nil)
}

func TestBuildEventQuery_Basic(t *testing.T) {
	testID := uuid.Must(uuid.NewV4())
	publisher := "flow-manager"

	query, args := buildEventQuery(publisher, testID, []string{"activeflow_*"}, "", 10)

	// Check that query contains expected parts
	if !strings.Contains(query, "SELECT timestamp, event_type, publisher, data_type, data") {
		t.Error("Query missing SELECT clause")
	}
	if !strings.Contains(query, "FROM events") {
		t.Error("Query missing FROM clause")
	}
	if !strings.Contains(query, "WHERE publisher = ?") {
		t.Error("Query missing publisher condition")
	}
	if !strings.Contains(query, "AND resource_id = ?") {
		t.Error("Query missing resource_id condition")
	}
	if !strings.Contains(query, "ORDER BY timestamp DESC LIMIT ?") {
		t.Error("Query missing ORDER BY and LIMIT clause")
	}

	// Check args
	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}
	if args[0] != publisher {
		t.Errorf("First arg should be publisher, got %v", args[0])
	}
	if args[1] != testID.String() {
		t.Errorf("Second arg should be resourceID, got %v", args[1])
	}
	if args[2] != 10 {
		t.Errorf("Third arg should be pageSize, got %v", args[2])
	}
}

func TestBuildEventQuery_WithPageToken(t *testing.T) {
	testID := uuid.Must(uuid.NewV4())
	publisher := "flow-manager"
	pageToken := "2024-01-15T10:29:00.123000Z"

	query, args := buildEventQuery(publisher, testID, []string{"activeflow_*"}, pageToken, 10)

	// Check that query contains pagination condition
	if !strings.Contains(query, "AND timestamp < ?") {
		t.Error("Query missing pagination condition")
	}

	// Check args include pageToken
	if len(args) != 4 {
		t.Errorf("Expected 4 args, got %d", len(args))
	}
	if args[2] != pageToken {
		t.Errorf("Third arg should be pageToken, got %v", args[2])
	}
}

func TestBuildEventQuery_WithEventFilters(t *testing.T) {
	testID := uuid.Must(uuid.NewV4())
	publisher := "flow-manager"

	query, _ := buildEventQuery(publisher, testID, []string{"activeflow_created", "flow_*"}, "", 10)

	// Check that query contains event conditions
	if !strings.Contains(query, "event_type = 'activeflow_created'") {
		t.Error("Query missing exact event filter")
	}
	if !strings.Contains(query, "event_type LIKE 'flow_%'") {
		t.Error("Query missing wildcard event filter")
	}
}

func TestBuildEventQuery_NoEventFilters(t *testing.T) {
	testID := uuid.Must(uuid.NewV4())
	publisher := "flow-manager"

	query, _ := buildEventQuery(publisher, testID, []string{}, "", 10)

	// Query should not have event type conditions
	if strings.Contains(query, "event_type =") || strings.Contains(query, "event_type LIKE") {
		t.Error("Query should not have event type conditions when events filter is empty")
	}
}

func TestBuildEventQuery_WildcardAll(t *testing.T) {
	testID := uuid.Must(uuid.NewV4())
	publisher := "flow-manager"

	query, _ := buildEventQuery(publisher, testID, []string{"*"}, "", 10)

	// Query should not have event type conditions when using "*"
	if strings.Contains(query, "event_type =") || strings.Contains(query, "event_type LIKE") {
		t.Error("Query should not have event type conditions when using '*' wildcard")
	}
}

func TestBuildEventQuery_ComplexScenario(t *testing.T) {
	testID := uuid.Must(uuid.NewV4())
	publisher := "call-manager"
	pageToken := "2024-01-15T10:29:00.123000Z"
	events := []string{"call_created", "call_hangup", "groupcall_*"}

	query, args := buildEventQuery(publisher, testID, events, pageToken, 50)

	// Verify all query parts are present
	if !strings.Contains(query, "event_type = 'call_created'") {
		t.Error("Query missing call_created filter")
	}
	if !strings.Contains(query, "event_type = 'call_hangup'") {
		t.Error("Query missing call_hangup filter")
	}
	if !strings.Contains(query, "event_type LIKE 'groupcall_%'") {
		t.Error("Query missing groupcall wildcard filter")
	}
	if !strings.Contains(query, "timestamp < ?") {
		t.Error("Query missing pagination condition")
	}

	// Verify args (publisher is passed as arg, not embedded in query)
	if len(args) != 4 {
		t.Errorf("Expected 4 args, got %d", len(args))
	}
	if args[0] != publisher {
		t.Errorf("First arg should be publisher (%s), got %v", publisher, args[0])
	}
	if args[1] != testID.String() {
		t.Errorf("Second arg should be resourceID, got %v", args[1])
	}
	if args[2] != pageToken {
		t.Errorf("Third arg should be pageToken, got %v", args[2])
	}
	if args[3] != 50 {
		t.Errorf("Last arg should be pageSize (50), got %v", args[3])
	}
}
