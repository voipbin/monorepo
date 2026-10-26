package requestexternal

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

const (
	testTwilioSID   = "AC00000000000000000000000000000000"
	testTwilioToken = "twilio-test-token-0000"
	testTrunkSID    = "TK00000000000000000000000000000000"
	testNumberSID   = "PN00000000000000000000000000000000"
)

type twilioSeen struct {
	method      string
	path        string
	rawQuery    string
	user        string
	pass        string
	hasAuth     bool
	contentType string
	body        string
}

// installTwilio starts a fake twilio, points the client to it and returns the requests it saw.
func installTwilio(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *[]twilioSeen {
	t.Helper()

	seen := []twilioSeen{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		u, p, ok := r.BasicAuth()
		seen = append(seen, twilioSeen{
			method:      r.Method,
			path:        r.URL.EscapedPath(),
			rawQuery:    r.URL.RawQuery,
			user:        u,
			pass:        p,
			hasAuth:     ok,
			contentType: r.Header.Get("Content-Type"),
			body:        string(b),
		})
		handler(w, r)
	}))

	prev := twilioBaseURL
	twilioBaseURL = srv.URL
	t.Cleanup(func() {
		twilioBaseURL = prev
		srv.Close()
	})

	return &seen
}

func reply(status int, body string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func Test_TwilioIncomingPhoneNumbersPost(t *testing.T) {
	seen := installTwilio(t, reply(http.StatusCreated, `{"sid":"`+testNumberSID+`","phone_number":"+15550000001","emergency_status":"Active"}`))

	res, err := NewRequestExternal().TwilioIncomingPhoneNumbersPost(testTwilioSID, testTwilioToken, "+15550000001", testTrunkSID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.SID != testNumberSID || res.EmergencyStatus != "Active" {
		t.Errorf("Wrong result. got: %+v", res)
	}

	if len(*seen) != 1 {
		t.Fatalf("Wrong request count. got: %d", len(*seen))
	}
	got := (*seen)[0]
	if got.method != http.MethodPost || got.path != "/2010-04-01/Accounts/"+testTwilioSID+"/IncomingPhoneNumbers.json" {
		t.Errorf("Wrong request line. got: %s %s", got.method, got.path)
	}
	if !got.hasAuth || got.user != testTwilioSID || got.pass != testTwilioToken {
		t.Errorf("Wrong basic auth.")
	}
	if got.contentType != "application/x-www-form-urlencoded" {
		t.Errorf("Wrong content type. got: %s", got.contentType)
	}
	form, _ := url.ParseQuery(got.body)
	if form.Get("PhoneNumber") != "+15550000001" || form.Get("TrunkSid") != testTrunkSID {
		t.Errorf("Wrong form. got: %s", got.body)
	}
	if !strings.Contains(got.body, "PhoneNumber=%2B155") {
		t.Errorf("The plus sign must be percent encoded. got: %s", got.body)
	}
}

func Test_TwilioIncomingPhoneNumbersPost_noTrunk(t *testing.T) {
	seen := installTwilio(t, reply(http.StatusOK, `{"sid":"`+testNumberSID+`"}`))

	if _, err := NewRequestExternal().TwilioIncomingPhoneNumbersPost(testTwilioSID, testTwilioToken, "+15550000001", ""); err != nil {
		t.Fatal(err)
	}
	if strings.Contains((*seen)[0].body, "TrunkSid") {
		t.Errorf("An empty trunk must not be sent. got: %s", (*seen)[0].body)
	}
}

func Test_TwilioIncomingPhoneNumbersPost_statusError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		expectCode int
	}{
		{"not available", http.StatusBadRequest, `{"code":21422,"message":"x","status":400}`, 21422},
		{"unauthorized", http.StatusUnauthorized, `{"code":20003,"message":"x","status":401}`, 20003},
		{"too many requests", http.StatusTooManyRequests, `{"code":20429}`, 20429},
		{"server error with a non-json body", http.StatusInternalServerError, `<html>boom</html>`, 0},
		{"accepted is not a success", http.StatusAccepted, `{"sid":"` + testNumberSID + `"}`, 0},
		{"no content is not a success for a purchase", http.StatusNoContent, ``, 0},
		{"redirect is not followed", http.StatusFound, ``, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seen := installTwilio(t, func(w http.ResponseWriter, r *http.Request) {
				if tt.status == http.StatusFound {
					w.Header().Set("Location", "/elsewhere")
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})

			_, err := NewRequestExternal().TwilioIncomingPhoneNumbersPost(testTwilioSID, testTwilioToken, "+15550000001", testTrunkSID)

			var statusErr *HTTPStatusError
			if !errors.As(err, &statusErr) || statusErr.StatusCode != tt.status || statusErr.ErrorCode != tt.expectCode {
				t.Errorf("Wrong error. expect: status %d code %d, got: %v", tt.status, tt.expectCode, err)
			}
			if len(*seen) != 1 {
				t.Errorf("Wrong request count. a redirect must not be followed. got: %d", len(*seen))
			}
		})
	}
}

func Test_TwilioIncomingPhoneNumbersPost_errorBodyIsBounded(t *testing.T) {
	big := `{"code":21422,"pad":"` + strings.Repeat("a", 3*maxTwilioErrorBody) + `"}`
	installTwilio(t, reply(http.StatusBadRequest, big))

	_, err := NewRequestExternal().TwilioIncomingPhoneNumbersPost(testTwilioSID, testTwilioToken, "+15550000001", "")

	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusBadRequest || statusErr.ErrorCode != 0 {
		t.Errorf("Wrong error. a body over the limit must not be parsed. got: %v", err)
	}
}

func Test_TwilioIncomingPhoneNumbersPost_parseError(t *testing.T) {
	installTwilio(t, reply(http.StatusCreated, `not json`))

	_, err := NewRequestExternal().TwilioIncomingPhoneNumbersPost(testTwilioSID, testTwilioToken, "+15550000001", "")
	var statusErr *HTTPStatusError
	if err == nil || errors.As(err, &statusErr) {
		t.Errorf("Wrong error. expect: a non status error, got: %v", err)
	}
}

func Test_Twilio_transportErrorHasNoSID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // connection refused

	prev := twilioBaseURL
	twilioBaseURL = url
	t.Cleanup(func() { twilioBaseURL = prev })

	for name, call := range map[string]func() error{
		"post": func() error {
			_, err := NewRequestExternal().TwilioIncomingPhoneNumbersPost(testTwilioSID, testTwilioToken, "+15550000001", "")
			return err
		},
		"delete": func() error {
			return NewRequestExternal().TwilioIncomingPhoneNumbersIDDelete(testTwilioSID, testTwilioToken, testNumberSID)
		},
		"search": func() error {
			_, err := NewRequestExternal().TwilioAvailableNumberGets(testTwilioSID, testTwilioToken, "US", 1)
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := call()
			if err == nil {
				t.Fatal("Wrong match. expect: error")
			}
			var statusErr *HTTPStatusError
			if errors.As(err, &statusErr) {
				t.Errorf("A transport error must not be a status error. got: %v", err)
			}
			if strings.Contains(err.Error(), testTwilioSID) || strings.Contains(err.Error(), testTwilioToken) || strings.Contains(err.Error(), "Accounts") {
				t.Errorf("The error leaks the URL or the credential. got: %v", err)
			}
		})
	}
}

func Test_TwilioIncomingPhoneNumbersIDDelete(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusOK} {
		seen := installTwilio(t, reply(status, ``))

		if err := NewRequestExternal().TwilioIncomingPhoneNumbersIDDelete(testTwilioSID, testTwilioToken, testNumberSID); err != nil {
			t.Fatalf("Wrong match. status: %d, got: %v", status, err)
		}
		got := (*seen)[0]
		if got.method != http.MethodDelete || got.path != "/2010-04-01/Accounts/"+testTwilioSID+"/IncomingPhoneNumbers/"+testNumberSID+".json" {
			t.Errorf("Wrong request line. got: %s %s", got.method, got.path)
		}
		if !got.hasAuth || got.user != testTwilioSID || got.pass != testTwilioToken {
			t.Errorf("Wrong basic auth.")
		}
	}

	installTwilio(t, reply(http.StatusNotFound, `{"code":20404}`))
	err := NewRequestExternal().TwilioIncomingPhoneNumbersIDDelete(testTwilioSID, testTwilioToken, testNumberSID)
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusNotFound || statusErr.ErrorCode != 20404 {
		t.Errorf("Wrong error. got: %v", err)
	}
}

func Test_TwilioIncomingPhoneNumbersIDDelete_pathIsEscaped(t *testing.T) {
	seen := installTwilio(t, reply(http.StatusNoContent, ``))

	_ = NewRequestExternal().TwilioIncomingPhoneNumbersIDDelete(testTwilioSID, testTwilioToken, "../Accounts/other")
	if strings.Contains((*seen)[0].path, "/../") || !strings.Contains((*seen)[0].path, "IncomingPhoneNumbers/") {
		t.Errorf("The id must stay inside its path segment. got: %s", (*seen)[0].path)
	}
}

func Test_TwilioAvailableNumberGets(t *testing.T) {
	body := `{"available_phone_numbers":[` +
		`{"phone_number":"+15550000001","iso_country":"US","region":"CA","postal_code":"94000","capabilities":{"voice":true,"SMS":true,"MMS":false,"fax":false}},` +
		`{"phone_number":"+15550000002","iso_country":"US"}]}`

	for limit, want := range map[uint]string{0: "10", 3: "3", 99: "10"} {
		seen := installTwilio(t, reply(http.StatusOK, body))

		res, err := NewRequestExternal().TwilioAvailableNumberGets(testTwilioSID, testTwilioToken, "US", limit)
		if err != nil || len(res) != 2 || res[0].PhoneNumber != "+15550000001" || !res[0].Capabilities.SMS {
			t.Fatalf("Wrong result. got: %+v, %v", res, err)
		}

		got := (*seen)[0]
		q, _ := url.ParseQuery(got.rawQuery)
		if got.method != http.MethodGet || got.path != "/2010-04-01/Accounts/"+testTwilioSID+"/AvailablePhoneNumbers/US/Local.json" {
			t.Errorf("Wrong request line. got: %s %s", got.method, got.path)
		}
		if q.Get("PageSize") != want || q.Get("VoiceEnabled") != "true" || q.Get("ExcludeAllAddressRequired") != "true" {
			t.Errorf("Wrong query. limit: %d, got: %s", limit, got.rawQuery)
		}
	}

	if _, err := NewRequestExternal().TwilioAvailableNumberGets(testTwilioSID, testTwilioToken, "", 1); err == nil {
		t.Errorf("Wrong match. expect: an error for no country")
	}

	installTwilio(t, reply(http.StatusOK, `not json`))
	if _, err := NewRequestExternal().TwilioAvailableNumberGets(testTwilioSID, testTwilioToken, "US", 1); err == nil {
		t.Errorf("Wrong match. expect: a parse error")
	}

	installTwilio(t, reply(http.StatusUnauthorized, `{"code":20003}`))
	_, err := NewRequestExternal().TwilioAvailableNumberGets(testTwilioSID, testTwilioToken, "US", 1)
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("Wrong error. got: %v", err)
	}
}

// the logs of a twilio request must never hold the credential or the account sid
func Test_Twilio_noCredentialInLogs(t *testing.T) {
	hook := logrustest.NewGlobal()
	prevLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() { logrus.SetLevel(prevLevel) })

	installTwilio(t, reply(http.StatusCreated, `{"sid":"`+testNumberSID+`"}`))
	_, _ = NewRequestExternal().TwilioIncomingPhoneNumbersPost(testTwilioSID, testTwilioToken, "+15550000001", testTrunkSID)
	_ = NewRequestExternal().TwilioIncomingPhoneNumbersIDDelete(testTwilioSID, testTwilioToken, testNumberSID)
	_, _ = NewRequestExternal().TwilioAvailableNumberGets(testTwilioSID, testTwilioToken, "US", 1)

	for _, e := range hook.AllEntries() {
		line := e.Message
		for _, v := range e.Data {
			line += " " + toString(v)
		}
		if strings.Contains(line, testTwilioToken) || strings.Contains(line, testTwilioSID) || strings.Contains(strings.ToLower(line), "authorization") {
			t.Errorf("A log entry holds a credential: %s", line)
		}
	}
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if e, ok := v.(error); ok {
		return e.Error()
	}
	return ""
}
