package transcript

import (
	"strings"
	"testing"
	"time"
)

// Test_CreateWebhookEvent_marshalError checks that an event that cannot be encoded is reported
// as an error and produces no payload. A year outside of 0..9999 cannot be written as JSON.
func Test_CreateWebhookEvent_marshalError(t *testing.T) {
	badTime := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)

	tr := &Transcript{TMCreate: &badTime}
	res, err := tr.CreateWebhookEvent()
	if err == nil || !strings.Contains(err.Error(), "year outside of range") {
		t.Errorf("Wrong match. expect: year range error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: no payload, got: %q", res)
	}
}
