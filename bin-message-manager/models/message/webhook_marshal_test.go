package message

import (
	"strings"
	"testing"
	"time"
)

// Test_CreateWebhookEvent_marshalFailure covers a message whose time value cannot be encoded as json
// (the year must be in [0,9999]).
func Test_CreateWebhookEvent_marshalFailure(t *testing.T) {
	tm := time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)
	m := &Message{TMCreate: &tm}

	res, err := m.CreateWebhookEvent()
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %s", res)
	}
	if err == nil || !strings.Contains(err.Error(), "year outside of range") {
		t.Errorf("Wrong match. expect: year outside of range error, got: %v", err)
	}
}
