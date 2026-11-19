package requestexternal

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

const (
	testTwilioSID      = "AC00000000000000000000000000000001"
	testTwilioToken    = "fake-twilio-token"
	testTwilioCallback = "https://hook.example.com/v1.0/messages/twilio"
)

func testTwilioConfig(sid string, token string, callback string) TwilioConfig {
	return TwilioConfig{
		Credentials: func(context.Context) (string, string, error) { return sid, token, nil },
		CallbackURL: func(context.Context) string { return callback },
	}
}

// twilioRecorder is a local server that records the requests that the Twilio sending makes and answers with a fixed
// status and body. Nothing leaves the machine.
type twilioRecorder struct {
	mu       sync.Mutex
	requests []*http.Request
	forms    []url.Values
	user     []string
	pass     []string
	paths    []string

	status int
	body   string
}

func useTwilioServer(t *testing.T, status int, body string) *twilioRecorder {
	t.Helper()

	rec := &twilioRecorder{status: status, body: body}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(raw))
		u, p, _ := r.BasicAuth()

		rec.mu.Lock()
		rec.requests = append(rec.requests, r)
		rec.forms = append(rec.forms, form)
		rec.user = append(rec.user, u)
		rec.pass = append(rec.pass, p)
		rec.paths = append(rec.paths, r.URL.Path)
		rec.mu.Unlock()

		w.WriteHeader(rec.status)
		_, _ = w.Write([]byte(rec.body))
	}))
	t.Cleanup(srv.Close)

	old := twilioAPIBase
	twilioAPIBase = srv.URL
	t.Cleanup(func() { twilioAPIBase = old })

	return rec
}

func Test_TwilioSendSMS_request(t *testing.T) {
	rec := useTwilioServer(t, http.StatusCreated, `{"sid":"SM00000000000000000000000000000001","num_segments":"2","status":"queued"}`)
	h := NewRequestExternal(staticToken("mb"), staticToken("tn"), testTwilioConfig(testTwilioSID, testTwilioToken, testTwilioCallback))

	res, err := h.TwilioSendSMS(context.Background(), "+15550000001", "+15550000002", "hello")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.SID != "SM00000000000000000000000000000001" || res.Segments() != 2 {
		t.Errorf("Wrong match. got: %v, segments: %d", res, res.Segments())
	}

	if len(rec.forms) != 1 {
		t.Fatalf("Wrong match. expect: one request, got: %d", len(rec.forms))
	}
	if rec.paths[0] != "/2010-04-01/Accounts/"+testTwilioSID+"/Messages.json" {
		t.Errorf("Wrong path. got: %s", rec.paths[0])
	}
	if rec.user[0] != testTwilioSID || rec.pass[0] != testTwilioToken {
		t.Errorf("Wrong basic auth.")
	}

	form := rec.forms[0]
	if form.Get("To") != "+15550000002" || form.Get("From") != "+15550000001" || form.Get("Body") != "hello" {
		t.Errorf("Wrong form. got: %v", form)
	}
	if form.Get("StatusCallback") != testTwilioCallback {
		t.Errorf("Wrong status callback. got: %v", form.Get("StatusCallback"))
	}
	if _, ok := form["MediaUrl"]; ok || form.Get("SendAsMms") != "" {
		t.Errorf("An sms must have no media and no SendAsMms. got: %v", form)
	}
}

func Test_TwilioSendMMS_request(t *testing.T) {
	tests := []struct {
		name  string
		media []string

		expectMedia  []string
		expectAsMMS  string
		expectNoBody bool
	}{
		{"with media", []string{"https://example.invalid/a.png", "https://example.invalid/b.png"}, []string{"https://example.invalid/a.png", "https://example.invalid/b.png"}, "", false},
		{"text only mms is sent as mms", nil, nil, "true", false},
		{"media without text", []string{"https://example.invalid/a.png"}, []string{"https://example.invalid/a.png"}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := useTwilioServer(t, http.StatusCreated, `{"sid":"MM00000000000000000000000000000001","num_segments":"1","status":"queued"}`)
			h := NewRequestExternal(staticToken("mb"), staticToken("tn"), testTwilioConfig(testTwilioSID, testTwilioToken, ""))

			text := "long text"
			if tt.expectNoBody {
				text = ""
			}
			if _, err := h.TwilioSendMMS(context.Background(), "+15550000001", "+15550000002", text, tt.media); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			form := rec.forms[0]
			if strings.Join(form["MediaUrl"], "|") != strings.Join(tt.expectMedia, "|") {
				t.Errorf("Wrong media. got: %v", form["MediaUrl"])
			}
			if form.Get("SendAsMms") != tt.expectAsMMS {
				t.Errorf("Wrong SendAsMms. expect: %q, got: %q", tt.expectAsMMS, form.Get("SendAsMms"))
			}
			if _, ok := form["Body"]; ok == tt.expectNoBody {
				t.Errorf("Wrong body presence. got: %v", form)
			}
			if _, ok := form["StatusCallback"]; ok {
				t.Errorf("The callback is not set, so no StatusCallback must be sent. got: %v", form)
			}
		})
	}
}

func Test_twilioSend_outcomes(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string

		expectOK        bool
		expectRejection bool
		expectSegments  int
		expectDetail    string
	}{
		{"created", 201, `{"sid":"SM00000000000000000000000000000001","num_segments":"3"}`, true, false, 3, ""},
		{"zero segments are not a count", 201, `{"sid":"SM00000000000000000000000000000001","num_segments":"0"}`, true, false, 0, ""},
		{"segments as a number", 200, `{"sid":"SM00000000000000000000000000000001","num_segments":4}`, true, false, 4, ""},
		{"bad request is a rejection", 400, `{"code":21211,"message":"The 'To' number is not valid","more_info":"x","status":400}`, false, true, 0, "code 21211 The 'To' number is not valid"},
		{"unauthorized is a rejection", 401, `{"code":20003,"message":"Authenticate","status":401}`, false, true, 0, "code 20003 Authenticate"},
		{"rate limit is a rejection", 429, `{"code":20429,"message":"Too many requests","status":429}`, false, true, 0, "code 20429 Too many requests"},
		{"408 is not a rejection", 408, ``, false, false, 0, ""},
		{"server error is not a rejection", 503, `{"code":20503,"message":"down"}`, false, false, 0, ""},
		{"unreadable answer is not known", 201, `not json`, false, false, 0, ""},
		{"answer without an id is not known", 201, `{"status":"queued"}`, false, false, 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useTwilioServer(t, tt.status, tt.body)
			h := NewRequestExternal(staticToken("mb"), staticToken("tn"), testTwilioConfig(testTwilioSID, testTwilioToken, ""))

			res, err := h.TwilioSendSMS(context.Background(), "+15550000001", "+15550000002", "hello")
			if tt.expectOK {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				if res.Segments() != tt.expectSegments {
					t.Errorf("Wrong segments. expect: %d, got: %d", tt.expectSegments, res.Segments())
				}
				return
			}

			if err == nil {
				t.Fatalf("Wrong match. expect: an error")
			}
			if IsDefinitiveRejection(err) != tt.expectRejection {
				t.Errorf("Wrong rejection. expect: %v, got: %v, err: %v", tt.expectRejection, IsDefinitiveRejection(err), err)
			}
			if tt.expectDetail != "" && !strings.Contains(err.Error(), tt.expectDetail) {
				t.Errorf("Wrong detail. expect: %s, got: %v", tt.expectDetail, err)
			}
			if strings.Contains(err.Error(), testTwilioSID) || strings.Contains(err.Error(), testTwilioToken) {
				t.Errorf("The error holds a credential: %v", err)
			}
		})
	}
}

// A transport error is not a rejection, and the error holds neither the request URL (its path holds the Account
// SID) nor the token.
func Test_twilioSend_transportError(t *testing.T) {
	old := twilioAPIBase
	twilioAPIBase = "http://127.0.0.1:1"
	t.Cleanup(func() { twilioAPIBase = old })

	h := NewRequestExternal(staticToken("mb"), staticToken("tn"), testTwilioConfig(testTwilioSID, testTwilioToken, ""))
	_, err := h.TwilioSendSMS(context.Background(), "+15550000001", "+15550000002", "hello")
	if err == nil || IsDefinitiveRejection(err) {
		t.Fatalf("Wrong match. expect: an error that is not a rejection, got: %v", err)
	}
	if strings.Contains(err.Error(), testTwilioSID) || strings.Contains(err.Error(), testTwilioToken) || strings.Contains(err.Error(), "Accounts") {
		t.Errorf("The error holds the request path or a credential: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = h.TwilioSendSMS(ctx, "+15550000001", "+15550000002", "hello")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Wrong match. expect: a context error that errors.Is can find, got: %v", err)
	}
}

// Without both credentials nothing is sent, and the zero config is the same as not configured.
func Test_twilio_notConfigured(t *testing.T) {
	rec := useTwilioServer(t, http.StatusCreated, `{"sid":"SM00000000000000000000000000000001"}`)

	tests := []struct {
		name   string
		config TwilioConfig
	}{
		{"zero config", TwilioConfig{}},
		{"no sid", testTwilioConfig("", testTwilioToken, "")},
		{"no token", testTwilioConfig(testTwilioSID, "", "")},
		{"neither", testTwilioConfig("", "", "")},
		{"credentials error", TwilioConfig{Credentials: func(context.Context) (string, string, error) { return "", "", errTestToken }}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewRequestExternal(staticToken("mb"), staticToken("tn"), tt.config)

			if h.TwilioAvailable(context.Background()) {
				t.Errorf("Wrong match. expect: not available")
			}
			_, err := h.TwilioSendSMS(context.Background(), "+15550000001", "+15550000002", "hello")
			if !IsTokenUnavailable(err) {
				t.Errorf("Wrong match. expect: token unavailable, got: %v", err)
			}
			if IsDefinitiveRejection(err) {
				t.Errorf("A request that was not sent is not a rejection: %v", err)
			}
		})
	}

	if len(rec.requests) != 0 {
		t.Errorf("Wrong match. expect: no request, got: %d", len(rec.requests))
	}

	h := NewRequestExternal(staticToken("mb"), staticToken("tn"), testTwilioConfig(testTwilioSID, testTwilioToken, ""))
	if !h.TwilioAvailable(context.Background()) {
		t.Errorf("Wrong match. expect: available")
	}
}

// Test_twilioSignature fixes the algorithm with a public vector that is independent of this code: the example of the
// Twilio helper libraries (token "12345", the request to https://mycompany.com/myapp.php?foo=1&bar=2 with five
// parameters), whose expected signature is published in their tests and in the Twilio security documentation.
func Test_twilioSignature(t *testing.T) {
	form := url.Values{
		"Digits":  {"1234"},
		"To":      {"+18005551212"},
		"From":    {"+14158675310"},
		"Caller":  {"+14158675310"},
		"CallSid": {"CA1234567890ABCDE"},
	}

	got := twilioSignature("12345", "https://mycompany.com/myapp.php?foo=1&bar=2", form)
	if got != "GvWf1cFY/Q7PnoempGyD5oXAezc=" {
		t.Errorf("Wrong signature for the public vector. got: %s", got)
	}
}

func Test_TwilioValidateCallback(t *testing.T) {
	form := url.Values{
		"MessageSid":    {"SM00000000000000000000000000000001"},
		"MessageStatus": {"delivered"},
		"To":            {"+15550000002"},
		"Empty":         {""},
		"Multi":         {"b", "a"},
		"Encoded":       {"a b&c=d"},
	}
	good := twilioSignature(testTwilioToken, testTwilioCallback, form)

	tests := []struct {
		name string

		config    TwilioConfig
		signature string
		form      url.Values

		expect error
	}{
		{"valid", testTwilioConfig(testTwilioSID, testTwilioToken, testTwilioCallback), good, form, nil},
		{"missing signature", testTwilioConfig(testTwilioSID, testTwilioToken, testTwilioCallback), "", form, ErrTwilioSignatureMissing},
		{"wrong signature", testTwilioConfig(testTwilioSID, testTwilioToken, testTwilioCallback), "AAAA" + good[4:], form, ErrTwilioSignatureInvalid},
		{"wrong token", testTwilioConfig(testTwilioSID, "other-token", testTwilioCallback), good, form, ErrTwilioSignatureInvalid},
		{"url with another scheme", testTwilioConfig(testTwilioSID, testTwilioToken, "http://hook.example.com/v1.0/messages/twilio"), good, form, ErrTwilioSignatureInvalid},
		{"url with a trailing slash", testTwilioConfig(testTwilioSID, testTwilioToken, testTwilioCallback+"/"), good, form, ErrTwilioSignatureInvalid},
		{"url with a port", testTwilioConfig(testTwilioSID, testTwilioToken, "https://hook.example.com:8443/v1.0/messages/twilio"), good, form, ErrTwilioSignatureInvalid},
		{"a changed value", testTwilioConfig(testTwilioSID, testTwilioToken, testTwilioCallback), good, url.Values{"MessageSid": {"SM00000000000000000000000000000001"}, "MessageStatus": {"failed"}}, ErrTwilioSignatureInvalid},
		{"an added parameter", testTwilioConfig(testTwilioSID, testTwilioToken, testTwilioCallback), good, func() url.Values {
			v := url.Values{}
			for k, vs := range form {
				v[k] = vs
			}
			v.Set("Extra", "1")
			return v
		}(), ErrTwilioSignatureInvalid},
		{"no token", testTwilioConfig(testTwilioSID, "", testTwilioCallback), good, form, ErrTwilioNotConfigured},
		{"no callback url", testTwilioConfig(testTwilioSID, testTwilioToken, ""), good, form, ErrTwilioNotConfigured},
		{"zero config", TwilioConfig{}, good, form, ErrTwilioNotConfigured},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewRequestExternal(staticToken("mb"), staticToken("tn"), tt.config)

			err := h.TwilioValidateCallback(context.Background(), tt.signature, tt.form)
			if !errors.Is(err, tt.expect) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, err)
			}
		})
	}
}

// The values of one key are sorted, a key with an empty value counts, and the percent decoded value is signed.
func Test_twilioSignature_forms(t *testing.T) {
	a, err := url.ParseQuery("B=2&A=1&A=0&C=")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	b := url.Values{"C": {""}, "A": {"0", "1"}, "B": {"2"}}
	if twilioSignature("t", "https://x.example/messages/twilio", a) != twilioSignature("t", "https://x.example/messages/twilio", b) {
		t.Errorf("The order of the keys and of the values of a key must not matter")
	}

	c, _ := url.ParseQuery("A=a%20b")
	d := url.Values{"A": {"a b"}}
	if twilioSignature("t", "u", c) != twilioSignature("t", "u", d) {
		t.Errorf("The decoded value must be signed")
	}
	if twilioSignature("t", "u", url.Values{"A": {""}}) == twilioSignature("t", "u", url.Values{}) {
		t.Errorf("A key with an empty value must count")
	}
}
