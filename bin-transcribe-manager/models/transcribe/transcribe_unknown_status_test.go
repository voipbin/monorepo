package transcribe

import (
	"strings"
	"testing"
	"time"
)

func Test_IsUpdatableStatus_unknownOldStatus(t *testing.T) {
	tests := []struct {
		name string

		oldStatus Status
		newStatus Status
	}{
		{"an unknown old status cannot move to done", Status("unknown"), StatusDone},
		{"an empty old status cannot move to progressing", Status(""), StatusProgressing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IsUpdatableStatus(tt.oldStatus, tt.newStatus) {
				t.Errorf("Wrong match. expect: false, got: true. old: %q, new: %q", tt.oldStatus, tt.newStatus)
			}
		})
	}
}

// Test_CreateWebhookEvent_marshalError checks that an event that cannot be encoded is reported
// as an error and produces no payload. A year outside of 0..9999 cannot be written as JSON.
func Test_CreateWebhookEvent_marshalError(t *testing.T) {
	badTime := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)

	tr := &Transcribe{TMCreate: &badTime}
	res, err := tr.CreateWebhookEvent()
	if err == nil || !strings.Contains(err.Error(), "year outside of range") {
		t.Errorf("Wrong match. expect: year range error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: no payload, got: %q", res)
	}
}
