package requestexternal

import (
	"net/http"
	"strings"
	"testing"
)

func Test_twilioSend_requestCannotBeBuilt(t *testing.T) {
	h := &requestExternal{}

	// a control character in the path makes the request URL invalid; no request is sent.
	res, err := h.twilioSend(http.MethodGet, testTwilioSID, testTwilioToken, "IncomingPhoneNumbers\x7f.json", nil, nil, http.StatusOK)
	if err == nil || !strings.Contains(err.Error(), "could not create a http request") {
		t.Errorf("Wrong match. expect error containing: could not create a http request, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %s", res)
	}
	// the account sid and token never appear in the error
	if err != nil && (strings.Contains(err.Error(), testTwilioSID) || strings.Contains(err.Error(), testTwilioToken)) {
		t.Errorf("Wrong match. the error carries a credential: %v", err)
	}
}

func Test_twilioSend_bodyCannotBeRead(t *testing.T) {
	// the answer announces more bytes than it sends, so the read of the body fails.
	seen := installTwilio(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("short"))
	})

	h := &requestExternal{}
	res, err := h.twilioSend(http.MethodGet, testTwilioSID, testTwilioToken, "IncomingPhoneNumbers.json", nil, nil, http.StatusOK)
	if err == nil || !strings.Contains(err.Error(), "could not read the response body") {
		t.Errorf("Wrong match. expect error containing: could not read the response body, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %s", res)
	}
	if len(*seen) != 1 {
		t.Errorf("Wrong match. expect: one request, got: %d", len(*seen))
	}
}
