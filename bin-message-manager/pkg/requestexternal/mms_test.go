package requestexternal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func Test_TelnyxSendMMS_request(t *testing.T) {
	tests := []struct {
		name string

		text string

		expectBody string
	}{
		{
			name:       "with text",
			text:       "hello",
			expectBody: `{"from":"+155****0001","to":"+155****0002","text":"hello","type":"MMS","media_urls":["https://example.invalid/a.png","https://example.invalid/b.pdf"]}`,
		},
		{
			name:       "media only omits the text",
			text:       "",
			expectBody: `{"from":"+155****0001","to":"+155****0002","type":"MMS","media_urls":["https://example.invalid/a.png","https://example.invalid/b.pdf"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotURL, gotAuth, gotContentType, gotBody string

			replaceDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
				b, _ := io.ReadAll(req.Body)
				gotMethod = req.Method
				gotURL = req.URL.String()
				gotAuth = req.Header.Get("Authorization")
				gotContentType = req.Header.Get("Content-Type")
				gotBody = string(b)

				return newResponse(200, io.NopCloser(strings.NewReader(`{"data":{"id":"fake-message-id","record_type":"message","type":"MMS"}}`))), nil
			}))

			h := NewRequestExternal(staticToken("fake-token-mb"), staticToken("fake-token-tn"))

			res, err := h.TelnyxSendMMS(context.Background(), "+155****0001", "+155****0002", tt.text, []string{"https://example.invalid/a.png", "https://example.invalid/b.pdf"})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if gotMethod != http.MethodPost || gotURL != "https://api.telnyx.com/v2/messages" {
				t.Errorf("Wrong request. method: %s, url: %s", gotMethod, gotURL)
			}
			if gotAuth != "Bearer fake-token-tn" || gotContentType != "application/json" {
				t.Errorf("Wrong headers. authorization: %s, content-type: %s", gotAuth, gotContentType)
			}
			if gotBody != tt.expectBody {
				t.Errorf("Wrong body.\nexpect: %s\ngot: %s", tt.expectBody, gotBody)
			}
			if res.Data.ID != "fake-message-id" {
				t.Errorf("Wrong response. got: %+v", res.Data)
			}
		})
	}
}

func Test_MessagebirdSendMMS_request(t *testing.T) {
	tests := []struct {
		name string

		text string

		expectBodyKeys map[string][]string
	}{
		{
			name: "with text",
			text: "hello",
			expectBodyKeys: map[string][]string{
				"recipients": {"+155****0002"},
				"originator": {"+155****0001"},
				"body":       {"hello"},
				"mediaUrls[]": {
					"https://example.invalid/a.png",
					"https://example.invalid/b.pdf",
				},
			},
		},
		{
			name: "media only omits the body",
			text: "",
			expectBodyKeys: map[string][]string{
				"recipients": {"+155****0002"},
				"originator": {"+155****0001"},
				"mediaUrls[]": {
					"https://example.invalid/a.png",
					"https://example.invalid/b.pdf",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotURL, gotAuth, gotContentType, gotBody string

			replaceDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
				b, _ := io.ReadAll(req.Body)
				gotMethod = req.Method
				gotURL = req.URL.String()
				gotAuth = req.Header.Get("Authorization")
				gotContentType = req.Header.Get("Content-Type")
				gotBody = string(b)

				return newResponse(201, io.NopCloser(strings.NewReader(`{"id":"fake-id","recipients":{"totalCount":1,"items":[{"recipient":155550002,"status":"sent"}]}}`))), nil
			}))

			h := NewRequestExternal(staticToken("fake-token-mb"), staticToken("fake-token-tn"))

			res, err := h.MessagebirdSendMMS(context.Background(), "+155****0001", "+155****0002", tt.text, []string{"https://example.invalid/a.png", "https://example.invalid/b.pdf"})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if gotMethod != http.MethodPost || gotURL != "https://rest.messagebird.com/mms" {
				t.Errorf("Wrong request. method: %s, url: %s", gotMethod, gotURL)
			}
			if gotAuth != "AccessKey fake-token-mb" || gotContentType != "application/x-www-form-urlencoded" {
				t.Errorf("Wrong headers. authorization: %s, content-type: %s", gotAuth, gotContentType)
			}

			values, errParse := url.ParseQuery(gotBody)
			if errParse != nil {
				t.Fatalf("Could not parse the body: %v", errParse)
			}
			if len(values) != len(tt.expectBodyKeys) {
				t.Errorf("Wrong keys. expect: %v, got: %v", tt.expectBodyKeys, values)
			}
			for k, expect := range tt.expectBodyKeys {
				got := values[k]
				if fmt.Sprint(got) != fmt.Sprint(expect) {
					t.Errorf("Wrong value for %s. expect: %v, got: %v", k, expect, got)
				}
			}
			if res.ID != "fake-id" {
				t.Errorf("Wrong response. got: %+v", res)
			}
		})
	}
}

func Test_SendMMS_status(t *testing.T) {
	tests := []struct {
		name string

		status int
		body   string

		expectErr        bool
		expectDefinitive bool
	}{
		{name: "200 is accepted", status: 200, body: `{}`},
		{name: "201 is accepted", status: 201, body: `{}`},
		{name: "300 is not accepted and not a rejection", status: 300, body: `{}`, expectErr: true},
		{name: "400 is a definitive rejection", status: 400, body: `{}`, expectErr: true, expectDefinitive: true},
		{name: "403 is a definitive rejection", status: 403, body: `{}`, expectErr: true, expectDefinitive: true},
		{name: "408 is ambiguous", status: 408, body: `{}`, expectErr: true},
		{name: "422 is a definitive rejection", status: 422, body: `{}`, expectErr: true, expectDefinitive: true},
		{name: "429 is a definitive rejection", status: 429, body: `{}`, expectErr: true, expectDefinitive: true},
		{name: "499 is a definitive rejection", status: 499, body: `{}`, expectErr: true, expectDefinitive: true},
		{name: "500 is ambiguous", status: 500, body: `{}`, expectErr: true},
		{name: "503 is ambiguous", status: 503, body: `{}`, expectErr: true},
	}

	for _, tt := range tests {
		for _, provider := range []string{"telnyx", "messagebird"} {
			t.Run(provider+" "+tt.name, func(t *testing.T) {
				// the body echoes a media URL, which must never reach the error text.
				body := `{"echo":"https://example.invalid/secret-media-url"}`
				if !tt.expectErr {
					body = tt.body
				}
				replaceDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
					return newResponse(tt.status, io.NopCloser(strings.NewReader(body))), nil
				}))

				h := NewRequestExternal(staticToken("fake-token-mb"), staticToken("fake-token-tn"))

				var err error
				if provider == "telnyx" {
					_, err = h.TelnyxSendMMS(context.Background(), "+155****0001", "+155****0002", "hello", []string{"https://example.invalid/secret-media-url"})
				} else {
					_, err = h.MessagebirdSendMMS(context.Background(), "+155****0001", "+155****0002", "hello", []string{"https://example.invalid/secret-media-url"})
				}

				if !tt.expectErr {
					if err != nil {
						t.Errorf("Wrong match. expect: ok, got: %v", err)
					}
					return
				}

				if err == nil {
					t.Fatalf("Wrong match. expect: error, got: nil")
				}
				if got := IsDefinitiveRejection(err); got != tt.expectDefinitive {
					t.Errorf("Wrong classification. expect: %v, got: %v, err: %v", tt.expectDefinitive, got, err)
				}
				if strings.Contains(err.Error(), "secret-media-url") {
					t.Errorf("The error text leaks the response body: %v", err)
				}
			})
		}
	}
}

func Test_SendMMS_notDefinitive(t *testing.T) {
	tests := []struct {
		name string

		roundTrip roundTripFunc
	}{
		{
			name: "transport failure",
			roundTrip: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("simulated transport failure")
			},
		},
		{
			name: "success status with a broken body",
			roundTrip: func(req *http.Request) (*http.Response, error) {
				return newResponse(200, io.NopCloser(strings.NewReader(`{bad`))), nil
			},
		},
		{
			name: "body read failure",
			roundTrip: func(req *http.Request) (*http.Response, error) {
				return newResponse(200, io.NopCloser(failingReader{})), nil
			},
		},
	}

	for _, tt := range tests {
		for _, provider := range []string{"telnyx", "messagebird"} {
			t.Run(provider+" "+tt.name, func(t *testing.T) {
				replaceDefaultTransport(t, tt.roundTrip)

				h := NewRequestExternal(staticToken("fake-token-mb"), staticToken("fake-token-tn"))

				var err error
				if provider == "telnyx" {
					_, err = h.TelnyxSendMMS(context.Background(), "+155****0001", "+155****0002", "hello", []string{"https://example.invalid/a.png"})
				} else {
					_, err = h.MessagebirdSendMMS(context.Background(), "+155****0001", "+155****0002", "hello", []string{"https://example.invalid/a.png"})
				}

				if err == nil {
					t.Fatalf("Wrong match. expect: error, got: nil")
				}
				if IsDefinitiveRejection(err) {
					t.Errorf("Wrong classification. a failure without a provider status must not be a definitive rejection: %v", err)
				}
			})
		}
	}
}

func Test_SendMMS_canceledContext(t *testing.T) {
	h := NewRequestExternal(staticToken("fake-token-mb"), staticToken("fake-token-tn"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// a canceled context fails before any network use, with a plain error that is not a rejection.
	replaceDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, req.Context().Err()
	}))

	if _, err := h.TelnyxSendMMS(ctx, "+155****0001", "+155****0002", "hello", []string{"https://example.invalid/a.png"}); !errors.Is(err, context.Canceled) || IsDefinitiveRejection(err) {
		t.Errorf("Wrong match. expect: canceled and not a rejection, got: %v", err)
	}
	if _, err := h.MessagebirdSendMMS(ctx, "+155****0001", "+155****0002", "hello", []string{"https://example.invalid/a.png"}); !errors.Is(err, context.Canceled) || IsDefinitiveRejection(err) {
		t.Errorf("Wrong match. expect: canceled and not a rejection, got: %v", err)
	}
}

func Test_IsDefinitiveRejection_nonProviderError(t *testing.T) {
	if IsDefinitiveRejection(nil) || IsDefinitiveRejection(errors.New("plain")) {
		t.Errorf("Wrong match. a nil or plain error is not a rejection")
	}
}

// Test_TelnyxSendMMS_textOnly pins that a text only mms (a promoted text) is sent without the media_urls key.
func Test_TelnyxSendMMS_textOnly(t *testing.T) {
	var gotBody string
	replaceDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(req.Body)
		gotBody = string(b)
		return newResponse(200, io.NopCloser(strings.NewReader(`{"data":{"id":"fake-message-id","parts":1}}`))), nil
	}))

	h := NewRequestExternal(staticToken("fake-token-mb"), staticToken("fake-token-tn"))
	res, err := h.TelnyxSendMMS(context.Background(), "+15550100001", "+15550100002", "long text", nil)
	if err != nil || res.Data.ID != "fake-message-id" {
		t.Fatalf("Wrong match. expect: ok, got: res: %v, err: %v", res, err)
	}

	expect := `{"from":"+15550100001","to":"+15550100002","text":"long text","type":"MMS"}`
	if gotBody != expect {
		t.Errorf("Wrong body.\nexpect: %s\ngot: %s", expect, gotBody)
	}
}
