package listenhandler

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"monorepo/bin-common-handler/models/sock"
)

func Test_redactSensitiveJSON(t *testing.T) {

	tests := []struct {
		name string
		data []byte

		expectRes []byte
	}{
		{
			name: "normal, top level password field is redacted",
			data: []byte(`{"customer_id":"7c1a1c3e-1111-11ea-af75-3f1e61b9a236","name":"ext-1000","password":"s3cr3t-plaintext"}`),

			expectRes: []byte(`{"customer_id":"7c1a1c3e-1111-11ea-af75-3f1e61b9a236","name":"ext-1000","password":"[REDACTED]"}`),
		},
		{
			name: "sensitive keys are matched case-insensitively",
			data: []byte(`{"Password":"a","SECRET":"b","Token":"c","credential":"d","name":"unchanged"}`),

			expectRes: []byte(`{"Password":"[REDACTED]","SECRET":"[REDACTED]","Token":"[REDACTED]","credential":"[REDACTED]","name":"unchanged"}`),
		},
		{
			name: "direct_hash field is redacted (extension direct-dial bearer secret)",
			data: []byte(`{"id":"7c1a1c3e-1111-11ea-af75-3f1e61b9a236","direct_id":"aaaa","direct_hash":"plaintext-direct-hash"}`),

			expectRes: []byte(`{"id":"7c1a1c3e-1111-11ea-af75-3f1e61b9a236","direct_id":"aaaa","direct_hash":"[REDACTED]"}`),
		},
		{
			name: "nested object password field is redacted",
			data: []byte(`{"trunk":{"username":"carrier1","password":"nested-secret"},"name":"trunk-1"}`),

			expectRes: []byte(`{"name":"trunk-1","trunk":{"password":"[REDACTED]","username":"carrier1"}}`),
		},
		{
			name: "password inside array elements is redacted",
			data: []byte(`[{"password":"a"},{"password":"b","name":"keep"}]`),

			expectRes: []byte(`[{"password":"[REDACTED]"},{"name":"keep","password":"[REDACTED]"}]`),
		},
		{
			name: "no sensitive fields, data unchanged",
			data: []byte(`{"customer_id":"7c1a1c3e-1111-11ea-af75-3f1e61b9a236","extension":"1000"}`),

			expectRes: []byte(`{"customer_id":"7c1a1c3e-1111-11ea-af75-3f1e61b9a236","extension":"1000"}`),
		},
		{
			name: "empty data returned as is",
			data: []byte{},

			expectRes: []byte{},
		},
		{
			name: "nil data returned as is",
			data: nil,

			expectRes: nil,
		},
		{
			name: "malformed json falls back to a fixed placeholder, never the raw bytes",
			data: []byte(`{"password":"leaked-if-not-careful"`),

			expectRes: []byte(`"[unparseable body redacted]"`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := redactSensitiveJSON(tt.data)

			if len(tt.expectRes) == 0 {
				if len(res) != 0 {
					t.Errorf("Wrong match. expect: empty, got: %s", res)
				}
				return
			}

			// compare as decoded JSON so key ordering differences don't fail the test
			var resDecoded, expectDecoded any
			if err := json.Unmarshal(res, &resDecoded); err != nil {
				t.Fatalf("redactSensitiveJSON returned invalid JSON: %v, res: %s", err, res)
			}
			if err := json.Unmarshal(tt.expectRes, &expectDecoded); err != nil {
				t.Fatalf("test expectation is not valid JSON: %v", err)
			}

			if !reflect.DeepEqual(expectDecoded, resDecoded) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectDecoded, resDecoded)
			}
		})
	}
}

func Test_redactSensitiveJSON_neverLeaksPlaintextPassword(t *testing.T) {
	data := []byte(`{"customer_id":"7c1a1c3e-1111-11ea-af75-3f1e61b9a236","name":"ext-1000","password":"MyS3cretSipPassw0rd!"}`)

	res := redactSensitiveJSON(data)

	if strings.Contains(string(res), "MyS3cretSipPassw0rd!") {
		t.Errorf("Plaintext password leaked into redacted output: %s", res)
	}
}

func Test_redactRequestForLog(t *testing.T) {

	tests := []struct {
		name string
		m    *sock.Request
	}{
		{
			name: "normal",
			m: &sock.Request{
				URI:    "/v1/extensions",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"password":"plaintext-secret"}`),
			},
		},
		{
			name: "nil request",
			m:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := redactRequestForLog(tt.m)

			if tt.m == nil {
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				return
			}

			if strings.Contains(string(res.Data), "plaintext-secret") {
				t.Errorf("Plaintext password leaked into redacted request: %s", res.Data)
			}
			if res.URI != tt.m.URI {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.m.URI, res.URI)
			}
			if res.Method != tt.m.Method {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.m.Method, res.Method)
			}

			// the original request passed in by the caller must be untouched
			if !strings.Contains(string(tt.m.Data), "plaintext-secret") {
				t.Errorf("Original request was mutated: %s", tt.m.Data)
			}
		})
	}
}

func Test_redactResponseForLog(t *testing.T) {

	tests := []struct {
		name string
		r    *sock.Response
	}{
		{
			name: "normal",
			r: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"password":"plaintext-secret"}`),
			},
		},
		{
			name: "nil response",
			r:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := redactResponseForLog(tt.r)

			if tt.r == nil {
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
				return
			}

			if strings.Contains(string(res.Data), "plaintext-secret") {
				t.Errorf("Plaintext password leaked into redacted response: %s", res.Data)
			}
			if res.StatusCode != tt.r.StatusCode {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.r.StatusCode, res.StatusCode)
			}

			// the original response passed in by the caller must be untouched
			if !strings.Contains(string(tt.r.Data), "plaintext-secret") {
				t.Errorf("Original response was mutated: %s", tt.r.Data)
			}
		})
	}
}
