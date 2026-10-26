package requestexternal

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"monorepo/bin-number-manager/pkg/requestexternal/models/telnyx"
)

const (
	testToken = "test-token"
	testAuth  = "Bearer test-token"
)

// unexpectedRequests counts requests that reached the fail-closed transport,
// that is requests no test case expected.
var unexpectedRequests atomic.Int64

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// failClosedTransport never reaches a network. It records the request and fails.
var failClosedTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
	unexpectedRequests.Add(1)
	return nil, errors.New("unexpected request: " + req.URL.String())
})

func TestMain(m *testing.M) {
	telnyxHTTPClient.Transport = failClosedTransport

	// a twilio test that forgets to install its server must never reach the real twilio
	twilioBaseURL = "http://127.0.0.1:0"
	os.Exit(m.Run())
}

type errBody struct{}

func (errBody) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (errBody) Close() error             { return nil }

// fakeReply describes what the fake provider answers.
type fakeReply struct {
	status       int
	body         string
	readErr      bool
	transportErr error
}

// recordedRequest is a request seen by the fake provider.
type recordedRequest struct {
	method      string
	url         string
	auth        string
	contentType string
	body        string
}

// installFake replaces the transport of the telnyx client and returns the
// recorded requests. The previous transport is restored and the number of
// requests is checked when the test ends.
func installFake(t *testing.T, wantRequests int, reply fakeReply) *[]recordedRequest {
	t.Helper()

	unexpectedRequests.Store(0)
	prev := telnyxHTTPClient.Transport
	got := []recordedRequest{}

	telnyxHTTPClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		rec := recordedRequest{
			method:      req.Method,
			url:         req.URL.String(),
			auth:        req.Header.Get("Authorization"),
			contentType: req.Header.Get("Content-Type"),
		}
		if req.Body != nil {
			b, _ := io.ReadAll(req.Body)
			rec.body = string(b)
		}
		got = append(got, rec)

		if reply.transportErr != nil {
			return nil, reply.transportErr
		}

		res := &http.Response{
			StatusCode: reply.status,
			Header:     http.Header{},
			Request:    req,
		}
		if reply.readErr {
			res.Body = errBody{}
		} else {
			res.Body = io.NopCloser(strings.NewReader(reply.body))
		}
		return res, nil
	})

	t.Cleanup(func() {
		telnyxHTTPClient.Transport = prev
		if len(got) != wantRequests {
			t.Errorf("Wrong request count. expect: %d, got: %d", wantRequests, len(got))
		}
		if n := unexpectedRequests.Load(); n != 0 {
			t.Errorf("Unexpected requests reached the fail-closed transport: %d", n)
		}
	})

	return &got
}

type apiCall struct {
	name string
	call func(h RequestExternal) (any, error)

	expectMethod      string
	expectURL         string
	expectContentType string
	expectBody        string

	okBody  string
	checkOK func(t *testing.T, res any)
}

func phoneNumberBody(id string) string {
	return `{"data":{"id":"` + id + `","phone_number":"+15550100001","status":"active","tags":["a"]}}`
}

func apiCalls() []apiCall {
	return []apiCall{
		{
			name: "TelnyxAvailableNumberGets",
			call: func(h RequestExternal) (any, error) {
				return h.TelnyxAvailableNumberGets(testToken, "US", "", "", 5)
			},
			expectMethod: http.MethodGet,
			expectURL:    "https://api.telnyx.com/v2/available_phone_numbers?filter[country_code]=US&filter[limit]=5",
			okBody:       `{"data":[{"phone_number":"+15550100001"},{"phone_number":"+15550100002"}]}`,
			checkOK: func(t *testing.T, res any) {
				got := res.([]*telnyx.AvailableNumber)
				if len(got) != 2 || got[0].PhoneNumber != "+15550100001" || got[1].PhoneNumber != "+15550100002" {
					t.Errorf("Wrong result. got: %+v", got)
				}
			},
		},
		{
			name: "TelnyxNumberOrdersPost",
			call: func(h RequestExternal) (any, error) {
				return h.TelnyxNumberOrdersPost(testToken, []string{"+15550100001", "+15550100002"}, "conn-1", "prof-1")
			},
			expectMethod:      http.MethodPost,
			expectURL:         "https://api.telnyx.com/v2/number_orders",
			expectContentType: "application/json",
			expectBody:        `{"phone_numbers":[{"phone_number":"+15550100001"},{"phone_number":"+15550100002"}],"connection_id":"conn-1","messaging_profile_id":"prof-1"}`,
			okBody:            `{"data":{"id":"order-1","status":"pending","phone_numbers":[{"phone_number":"+15550100001"}]}}`,
			checkOK: func(t *testing.T, res any) {
				got := res.(*telnyx.OrderNumber)
				if got.ID != "order-1" || len(got.PhoneNumbers) != 1 || got.PhoneNumbers[0].PhoneNumber != "+15550100001" {
					t.Errorf("Wrong result. got: %+v", got)
				}
			},
		},
		{
			name: "TelnyxPhoneNumbersGetByNumber",
			call: func(h RequestExternal) (any, error) {
				return h.TelnyxPhoneNumbersGetByNumber(testToken, "+15550100001")
			},
			expectMethod: http.MethodGet,
			expectURL:    "https://api.telnyx.com/v2/phone_numbers?filter[phone_number]=+15550100001",
			okBody:       `{"data":[{"id":"pn-1","phone_number":"+15550100001","status":"active"}]}`,
			checkOK: func(t *testing.T, res any) {
				got := res.(*telnyx.PhoneNumber)
				if got.ID != "pn-1" || got.Status != telnyx.PhoneNumberStatusActive {
					t.Errorf("Wrong result. got: %+v", got)
				}
			},
		},
		{
			name: "TelnyxPhoneNumbersIDGet",
			call: func(h RequestExternal) (any, error) {
				return h.TelnyxPhoneNumbersIDGet(testToken, "pn-1")
			},
			expectMethod: http.MethodGet,
			expectURL:    "https://api.telnyx.com/v2/phone_numbers/pn-1",
			okBody:       phoneNumberBody("pn-1"),
			checkOK: func(t *testing.T, res any) {
				got := res.(*telnyx.PhoneNumber)
				if got.ID != "pn-1" || got.PhoneNumber != "+15550100001" {
					t.Errorf("Wrong result. got: %+v", got)
				}
			},
		},
		{
			name: "TelnyxPhoneNumbersIDDelete",
			call: func(h RequestExternal) (any, error) {
				return h.TelnyxPhoneNumbersIDDelete(testToken, "pn-1")
			},
			expectMethod: http.MethodDelete,
			expectURL:    "https://api.telnyx.com/v2/phone_numbers/pn-1",
			okBody:       phoneNumberBody("pn-1"),
			checkOK: func(t *testing.T, res any) {
				got := res.(*telnyx.PhoneNumber)
				if got.ID != "pn-1" {
					t.Errorf("Wrong result. got: %+v", got)
				}
			},
		},
		{
			name: "TelnyxPhoneNumbersGet",
			call: func(h RequestExternal) (any, error) {
				return h.TelnyxPhoneNumbersGet(testToken, 3, "tag-1", "+15550100001")
			},
			expectMethod: http.MethodGet,
			expectURL:    "https://api.telnyx.com/v2/phone_numbers?page[size]=3&filter[tag]=tag-1&filter[phone_number]=+15550100001",
			okBody:       `{"data":[{"id":"pn-1"},{"id":"pn-2"}]}`,
			checkOK: func(t *testing.T, res any) {
				got := res.([]*telnyx.PhoneNumber)
				if len(got) != 2 || got[0].ID != "pn-1" || got[1].ID != "pn-2" {
					t.Errorf("Wrong result. got: %+v", got)
				}
			},
		},
		{
			name: "TelnyxPhoneNumbersIDUpdate",
			call: func(h RequestExternal) (any, error) {
				return h.TelnyxPhoneNumbersIDUpdate(testToken, "pn-1", map[string]interface{}{"tags": []string{"a", "b"}})
			},
			expectMethod:      http.MethodPatch,
			expectURL:         "https://api.telnyx.com/v2/phone_numbers/pn-1",
			expectContentType: "application/json",
			expectBody:        `{"tags":["a","b"]}`,
			okBody:            phoneNumberBody("pn-1"),
			checkOK: func(t *testing.T, res any) {
				got := res.(*telnyx.PhoneNumber)
				if got.ID != "pn-1" || len(got.Tags) != 1 || got.Tags[0] != "a" {
					t.Errorf("Wrong result. got: %+v", got)
				}
			},
		},
	}
}

func Test_Telnyx_success(t *testing.T) {
	for _, tt := range apiCalls() {
		t.Run(tt.name, func(t *testing.T) {
			reqs := installFake(t, 1, fakeReply{status: http.StatusOK, body: tt.okBody})

			res, err := tt.call(NewRequestExternal())
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			tt.checkOK(t, res)

			if len(*reqs) != 1 {
				return
			}
			got := (*reqs)[0]
			if got.method != tt.expectMethod {
				t.Errorf("Wrong method. expect: %s, got: %s", tt.expectMethod, got.method)
			}
			if got.url != tt.expectURL {
				t.Errorf("Wrong url. expect: %s, got: %s", tt.expectURL, got.url)
			}
			if got.auth != testAuth {
				t.Errorf("Wrong authorization. expect: %s, got: %s", testAuth, got.auth)
			}
			if got.contentType != tt.expectContentType {
				t.Errorf("Wrong content type. expect: %q, got: %q", tt.expectContentType, got.contentType)
			}
			if got.body != tt.expectBody {
				t.Errorf("Wrong body.\nexpect: %s\ngot: %s", tt.expectBody, got.body)
			}
		})
	}
}

func Test_Telnyx_failure(t *testing.T) {
	failures := []struct {
		name         string
		reply        func(okBody string) fakeReply
		expectSubstr string
	}{
		{
			name:         "provider rejects with status 404",
			reply:        func(string) fakeReply { return fakeReply{status: http.StatusNotFound, body: `{"errors":[]}`} },
			expectSubstr: "status: 404",
		},
		{
			name:         "provider answers server error",
			reply:        func(string) fakeReply { return fakeReply{status: http.StatusInternalServerError} },
			expectSubstr: "status: 500",
		},
		{
			name:         "status 201 is not accepted",
			reply:        func(ok string) fakeReply { return fakeReply{status: http.StatusCreated, body: ok} },
			expectSubstr: "status: 201",
		},
		{
			name: "transport error",
			reply: func(string) fakeReply {
				return fakeReply{transportErr: errors.New("connection refused")}
			},
			expectSubstr: "connection refused",
		},
		{
			name:         "body read error",
			reply:        func(string) fakeReply { return fakeReply{status: http.StatusOK, readErr: true} },
			expectSubstr: "could not read the response body",
		},
		{
			name:         "malformed json",
			reply:        func(string) fakeReply { return fakeReply{status: http.StatusOK, body: `{"data":`} },
			expectSubstr: "unexpected end of JSON input",
		},
	}

	for _, ac := range apiCalls() {
		for _, f := range failures {
			t.Run(ac.name+"/"+f.name, func(t *testing.T) {
				installFake(t, 1, f.reply(ac.okBody))

				res, err := ac.call(NewRequestExternal())
				if err == nil {
					t.Fatalf("Wrong match. expect: error, got: ok (%v)", res)
				}
				if !strings.Contains(err.Error(), f.expectSubstr) {
					t.Errorf("Wrong error. expect substring: %q, got: %v", f.expectSubstr, err)
				}
			})
		}
	}
}

func Test_TelnyxAvailableNumberGets_query(t *testing.T) {
	tests := []struct {
		name string

		countryCode        string
		locality           string
		administrativeArea string
		limit              uint

		expectURL string
	}{
		{"limit zero falls back to 10", "US", "", "", 0, "https://api.telnyx.com/v2/available_phone_numbers?filter[country_code]=US&filter[limit]=10"},
		{"limit one is kept", "US", "", "", 1, "https://api.telnyx.com/v2/available_phone_numbers?filter[country_code]=US&filter[limit]=1"},
		{"limit ten is kept", "US", "", "", 10, "https://api.telnyx.com/v2/available_phone_numbers?filter[country_code]=US&filter[limit]=10"},
		{"limit eleven is clamped to 10", "US", "", "", 11, "https://api.telnyx.com/v2/available_phone_numbers?filter[country_code]=US&filter[limit]=10"},
		{"locality only", "US", "Springfield", "", 3, "https://api.telnyx.com/v2/available_phone_numbers?filter[country_code]=US&filter[locality]=Springfield&filter[limit]=3"},
		{"administrative area only", "US", "", "IL", 3, "https://api.telnyx.com/v2/available_phone_numbers?filter[country_code]=US&filter[administrative_area]=IL&filter[limit]=3"},
		{"locality and administrative area", "US", "Springfield", "IL", 3, "https://api.telnyx.com/v2/available_phone_numbers?filter[country_code]=US&filter[locality]=Springfield&filter[administrative_area]=IL&filter[limit]=3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqs := installFake(t, 1, fakeReply{status: http.StatusOK, body: `{"data":[]}`})

			res, err := NewRequestExternal().TelnyxAvailableNumberGets(testToken, tt.countryCode, tt.locality, tt.administrativeArea, tt.limit)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || len(res) != 0 {
				t.Errorf("Wrong result. expect: empty non-nil slice, got: %v", res)
			}
			if len(*reqs) == 1 && (*reqs)[0].url != tt.expectURL {
				t.Errorf("Wrong url.\nexpect: %s\ngot: %s", tt.expectURL, (*reqs)[0].url)
			}
		})
	}
}

func Test_TelnyxAvailableNumberGets_noCountry(t *testing.T) {
	installFake(t, 0, fakeReply{status: http.StatusOK})

	if _, err := NewRequestExternal().TelnyxAvailableNumberGets(testToken, "", "", "", 5); err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_TelnyxPhoneNumbersGet_query(t *testing.T) {
	tests := []struct {
		name string

		size   uint
		tag    string
		number string

		expectURL string
	}{
		{"size zero falls back to 10", 0, "", "", "https://api.telnyx.com/v2/phone_numbers?page[size]=10"},
		{"size one is kept", 1, "", "", "https://api.telnyx.com/v2/phone_numbers?page[size]=1"},
		{"tag only", 5, "tag-1", "", "https://api.telnyx.com/v2/phone_numbers?page[size]=5&filter[tag]=tag-1"},
		{"number only", 5, "", "+15550100001", "https://api.telnyx.com/v2/phone_numbers?page[size]=5&filter[phone_number]=+15550100001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqs := installFake(t, 1, fakeReply{status: http.StatusOK, body: `{"data":[]}`})

			res, err := NewRequestExternal().TelnyxPhoneNumbersGet(testToken, tt.size, tt.tag, tt.number)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || len(res) != 0 {
				t.Errorf("Wrong result. expect: empty non-nil slice, got: %v", res)
			}
			if len(*reqs) == 1 && (*reqs)[0].url != tt.expectURL {
				t.Errorf("Wrong url.\nexpect: %s\ngot: %s", tt.expectURL, (*reqs)[0].url)
			}
		})
	}
}

func Test_TelnyxPhoneNumbersGetByNumber_noData(t *testing.T) {
	installFake(t, 1, fakeReply{status: http.StatusOK, body: `{"data":[]}`})

	res, err := NewRequestExternal().TelnyxPhoneNumbersGetByNumber(testToken, "+15550100001")
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: ok (%v)", res)
	}
	if !strings.Contains(err.Error(), "no phone number found for number: +15550100001") {
		t.Errorf("Wrong error. got: %v", err)
	}
}

func Test_TelnyxPhoneNumbersGetByNumber_firstOfMany(t *testing.T) {
	installFake(t, 1, fakeReply{status: http.StatusOK, body: `{"data":[{"id":"pn-1"},{"id":"pn-2"}]}`})

	res, err := NewRequestExternal().TelnyxPhoneNumbersGetByNumber(testToken, "+15550100001")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != "pn-1" {
		t.Errorf("Wrong result. expect: pn-1, got: %s", res.ID)
	}
}

func Test_TelnyxPhoneNumbersGet_distinctItems(t *testing.T) {
	installFake(t, 1, fakeReply{status: http.StatusOK, body: `{"data":[{"id":"pn-1"},{"id":"pn-2"},{"id":"pn-3"}]}`})

	res, err := NewRequestExternal().TelnyxPhoneNumbersGet(testToken, 3, "", "")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	ids := []string{}
	for _, tmp := range res {
		ids = append(ids, tmp.ID)
	}
	if strings.Join(ids, ",") != "pn-1,pn-2,pn-3" {
		t.Errorf("Wrong result. expect: pn-1,pn-2,pn-3, got: %v", ids)
	}
}

func Test_TelnyxPhoneNumbersIDUpdate_nilData(t *testing.T) {
	reqs := installFake(t, 1, fakeReply{status: http.StatusOK, body: phoneNumberBody("pn-1")})

	if _, err := NewRequestExternal().TelnyxPhoneNumbersIDUpdate(testToken, "pn-1", nil); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(*reqs) == 1 && (*reqs)[0].body != "null" {
		t.Errorf("Wrong body. expect: null, got: %s", (*reqs)[0].body)
	}
}

func Test_TelnyxPhoneNumbersIDUpdate_unmarshalableData(t *testing.T) {
	installFake(t, 0, fakeReply{status: http.StatusOK})

	_, err := NewRequestExternal().TelnyxPhoneNumbersIDUpdate(testToken, "pn-1", map[string]interface{}{"bad": make(chan int)})
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
}

func Test_Telnyx_invalidURLInput(t *testing.T) {
	const bad = "\x7f"

	tests := []struct {
		name string
		call func(h RequestExternal) (any, error)
	}{
		{"available numbers country code", func(h RequestExternal) (any, error) {
			return h.TelnyxAvailableNumberGets(testToken, bad, "", "", 1)
		}},
		{"phone numbers by number", func(h RequestExternal) (any, error) {
			return h.TelnyxPhoneNumbersGetByNumber(testToken, bad)
		}},
		{"phone numbers id get", func(h RequestExternal) (any, error) {
			return h.TelnyxPhoneNumbersIDGet(testToken, bad)
		}},
		{"phone numbers id delete", func(h RequestExternal) (any, error) {
			return h.TelnyxPhoneNumbersIDDelete(testToken, bad)
		}},
		{"phone numbers id update", func(h RequestExternal) (any, error) {
			return h.TelnyxPhoneNumbersIDUpdate(testToken, bad, map[string]interface{}{})
		}},
		{"phone numbers tag", func(h RequestExternal) (any, error) {
			return h.TelnyxPhoneNumbersGet(testToken, 1, bad, "")
		}},
		{"phone numbers number", func(h RequestExternal) (any, error) {
			return h.TelnyxPhoneNumbersGet(testToken, 1, "", bad)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installFake(t, 0, fakeReply{status: http.StatusOK})

			_, err := tt.call(NewRequestExternal())
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: ok")
			}
			if !strings.Contains(err.Error(), "could not create a http request") {
				t.Errorf("Wrong error. got: %v", err)
			}
		})
	}
}
