package failedevent

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func TestFailedEvent_Struct(t *testing.T) {
	tmCreate := time.Date(2023, 6, 7, 3, 22, 17, 995000000, time.UTC)
	tmUpdate := time.Date(2023, 6, 8, 10, 15, 30, 500000000, time.UTC)
	nextRetry := time.Date(2023, 6, 9, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		event *FailedEvent
	}{
		{
			name: "full failed event data",
			event: &FailedEvent{
				ID:             uuid.FromStringOrNil("6ecdb856-0600-11ee-b746-d3ef5adc8ef7"),
				EventType:      "billing.account.created",
				EventPublisher: "billing-manager",
				EventData:      `{"id":"123","customer_id":"456"}`,
				ErrorMessage:   "database connection failed",
				RetryCount:     3,
				MaxRetries:     5,
				NextRetryAt:    &nextRetry,
				Status:         StatusRetrying,
				TMCreate:       &tmCreate,
				TMUpdate:       &tmUpdate,
			},
		},
		{
			name: "minimal failed event data",
			event: &FailedEvent{
				ID:             uuid.FromStringOrNil("9b219300-0600-11ee-bd0c-db57aac06783"),
				EventType:      "billing.event.failed",
				EventPublisher: "test-service",
				EventData:      "{}",
				ErrorMessage:   "",
				RetryCount:     0,
				MaxRetries:     3,
				Status:         StatusPending,
				TMCreate:       &tmCreate,
			},
		},
		{
			name: "exhausted failed event",
			event: &FailedEvent{
				ID:             uuid.FromStringOrNil("ab219300-0600-11ee-bd0c-db57aac06783"),
				EventType:      "billing.event.exhausted",
				EventPublisher: "test-service",
				EventData:      `{"error":"permanent failure"}`,
				ErrorMessage:   "maximum retries exceeded",
				RetryCount:     5,
				MaxRetries:     5,
				Status:         StatusExhausted,
				TMCreate:       &tmCreate,
				TMUpdate:       &tmUpdate,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.event.ID == uuid.Nil {
				t.Error("Expected non-nil ID")
			}

			if tt.event.EventType == "" {
				t.Error("Expected non-empty EventType")
			}

			if tt.event.EventPublisher == "" {
				t.Error("Expected non-empty EventPublisher")
			}

			if tt.event.TMCreate == nil {
				t.Error("Expected non-nil TMCreate")
			}
		})
	}
}

func TestStatus_Constants(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected string
	}{
		{"pending", StatusPending, "pending"},
		{"retrying", StatusRetrying, "retrying"},
		{"exhausted", StatusExhausted, "exhausted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("Status = %s, expected %s", tt.status, tt.expected)
			}
		})
	}
}
