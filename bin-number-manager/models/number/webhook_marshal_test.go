package number

import (
	"testing"
	"time"
)

func Test_CreateWebhookEvent_unencodable(t *testing.T) {
	// a time outside the range of the JSON encoder makes the marshal fail
	tm := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	n := &Number{TMCreate: &tm}

	res, err := n.CreateWebhookEvent()
	if err == nil {
		t.Errorf("Wrong match. expect: an error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %s", res)
	}
}
