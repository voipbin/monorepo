package email

import (
	"strings"
	"testing"
	"time"
)

func Test_CreateWebhookEvent_marshalFails(t *testing.T) {
	// year 10000 cannot be encoded as JSON
	tm := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	e := &Email{TMCreate: &tm}

	res, err := e.CreateWebhookEvent()
	if err == nil || !strings.Contains(err.Error(), "year outside of range") {
		t.Errorf("Wrong match. expect a marshal error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Expected a nil result. got: %s", string(res))
	}
}

func Test_CreateWebhookEvent_success(t *testing.T) {
	tm := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	e := &Email{Subject: "synthetic subject", TMCreate: &tm}

	res, err := e.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !strings.Contains(string(res), `"subject":"synthetic subject"`) || !strings.Contains(string(res), `"tm_create":"2026-01-02T03:04:05Z"`) {
		t.Errorf("Wrong webhook event. got: %s", string(res))
	}
}
