package requestexternal

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

func Test_TelnyxNumberOrdersPost_statusError(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnprocessableEntity, http.StatusTooManyRequests, http.StatusInternalServerError} {
		installFake(t, 1, fakeReply{status: status})

		_, err := NewRequestExternal().TelnyxNumberOrdersPost(testToken, []string{"+155****0001"}, "conn", "prof")

		var statusErr *HTTPStatusError
		if !errors.As(err, &statusErr) || statusErr.StatusCode != status {
			t.Errorf("Wrong error. expect: HTTPStatusError with status %d, got: %v", status, err)
		}
	}
}

// the logs of a request must never hold the credential of the request
func Test_TelnyxPhoneNumbersGetByNumber_noAuthorizationInLogs(t *testing.T) {
	hook := logrustest.NewGlobal()
	prevLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() {
		logrus.SetLevel(prevLevel)
		hook.Reset()
	})

	installFake(t, 1, fakeReply{status: http.StatusOK, body: `{"data":[{"id":"pn-1","phone_number":"+155****0001","status":"active"}]}`})

	if _, err := NewRequestExternal().TelnyxPhoneNumbersGetByNumber(testToken, "+155****0001"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	for _, e := range hook.AllEntries() {
		// the daemon logs JSON, which serializes the fields of a logged struct deeply
		line, _ := (&logrus.JSONFormatter{}).Format(e)
		text, _ := e.String()
		if strings.Contains(string(line), testToken) || strings.Contains(text, testToken) {
			t.Errorf("A log entry holds the credential: %s", line)
		}
	}
}

func Test_HTTPStatusError_message(t *testing.T) {
	if got := (&HTTPStatusError{StatusCode: 404}).Error(); got != "could not get correct response. status: 404" {
		t.Errorf("Wrong message. got: %s", got)
	}
	if got := (&HTTPStatusError{StatusCode: 400, ErrorCode: 21422}).Error(); got != "could not get correct response. status: 400, code: 21422" {
		t.Errorf("Wrong message. got: %s", got)
	}
}
