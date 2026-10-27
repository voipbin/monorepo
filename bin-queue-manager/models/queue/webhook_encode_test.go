package queue

import (
	"testing"
	"time"
)

func Test_CreateWebhookEvent_unencodableTime(t *testing.T) {
	// a timestamp outside of the range that JSON can express
	outOfRange := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)

	res, err := (&Queue{TMCreate: &outOfRange}).CreateWebhookEvent()
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %s", res)
	}
}
