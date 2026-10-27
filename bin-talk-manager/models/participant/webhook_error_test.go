package participant

import (
	"strings"
	"testing"
	"time"
)

// A time outside of the years 0 to 9999 cannot be encoded as JSON, so the webhook event of
// the participant must report the failure and carry no payload.
func Test_Participant_CreateWebhookEvent_error(t *testing.T) {
	outOfRange := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)

	e := &Participant{TMJoined: &outOfRange}

	res, err := e.CreateWebhookEvent()
	if err == nil || !strings.Contains(err.Error(), "year outside of range") {
		t.Errorf("error = %v, want the year range failure", err)
	}
	if res != nil {
		t.Errorf("result = %s, want nil", res)
	}
}
