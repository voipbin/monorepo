package failedevent

import (
	"time"

	"github.com/gofrs/uuid"
)

// FailedEvent represents a failed billing event persisted for retry.
type FailedEvent struct {
	ID             uuid.UUID  `json:"id" db:"id,uuid"`
	EventType      string     `json:"event_type" db:"event_type"`
	EventPublisher string     `json:"event_publisher" db:"event_publisher"`
	EventData      string     `json:"event_data" db:"event_data"`
	ErrorMessage   string     `json:"error_message" db:"error_message"`
	RetryCount     int        `json:"retry_count" db:"retry_count"`
	MaxRetries     int        `json:"max_retries" db:"max_retries"`
	NextRetryAt    *time.Time `json:"next_retry_at" db:"next_retry_at"`
	Status         Status     `json:"status" db:"status"`
	TMCreate       *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate       *time.Time `json:"tm_update" db:"tm_update"`
}

// Status represents the status of a failed event.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRetrying  Status = "retrying"
	StatusExhausted Status = "exhausted"
)
