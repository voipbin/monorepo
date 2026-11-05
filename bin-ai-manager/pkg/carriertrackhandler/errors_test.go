package carriertrackhandler

import (
	"strings"
	"testing"
	"time"

	"monorepo/bin-ai-manager/models/mcpserver"
)

var allCodes = []ErrCode{
	CodeCredentialNotConfigured, CodeCredentialMalformed, CodeCredentialInvalid, CodeRateLimited,
	CodeCarrierUnavailable, CodeServiceBusy, CodeInvalidFormat, CodeNotFound, CodeToolUnknown,
}

func Test_Message_fitsTheErrorTextLimit(t *testing.T) {
	for _, carrier := range []string{"ups", "fedex", "dhl", "fake"} {
		for _, format := range []mcpserver.CredentialFormat{mcpserver.CredentialClientIDSecret, mcpserver.CredentialAPIKey} {
			for _, code := range allCodes {
				text := Message(code, carrier, format)
				if text == "" {
					t.Errorf("%s/%s/%s: empty text", carrier, format, code)
				}
				if len(text) > 200 {
					t.Errorf("%s/%s/%s: %d bytes exceeds the 200 byte limit of the error text the model sees: %q", carrier, format, code, len(text), text)
				}
				if strings.ContainsAny(text, "{}") {
					t.Errorf("%s/%s/%s: unresolved placeholder in %q", carrier, format, code, text)
				}
			}
		}
	}
}

func Test_Message_content(t *testing.T) {
	tests := []struct {
		code    ErrCode
		carrier string
		format  mcpserver.CredentialFormat
		expect  string
	}{
		{CodeCredentialNotConfigured, "fedex", mcpserver.CredentialClientIDSecret, "No FedEx API key is saved"},
		{CodeCredentialMalformed, "ups", mcpserver.CredentialClientIDSecret, "CLIENT_ID:CLIENT_SECRET"},
		{CodeCredentialMalformed, "dhl", mcpserver.CredentialAPIKey, "invalid format"},
		{CodeCredentialInvalid, "dhl", mcpserver.CredentialAPIKey, "DHL rejected the saved API key"},
		{CodeInvalidFormat, "ups", mcpserver.CredentialClientIDSecret, "1Z"},
		{CodeInvalidFormat, "dhl", mcpserver.CredentialAPIKey, "does not look valid for DHL"},
		{CodeNotFound, "ups", mcpserver.CredentialClientIDSecret, "UPS has no record"},
		{CodeToolUnknown, "ups", mcpserver.CredentialClientIDSecret, "does not offer that tool"},
		{CodeServiceBusy, "fake", mcpserver.CredentialAPIKey, "for FAKE right now"},
	}
	for _, tt := range tests {
		if got := Message(tt.code, tt.carrier, tt.format); !strings.Contains(got, tt.expect) {
			t.Errorf("Message(%s, %s) = %q, expected to contain %q", tt.code, tt.carrier, got, tt.expect)
		}
	}
}

func Test_CarrierError_cooldown(t *testing.T) {
	tests := []struct {
		name   string
		err    *CarrierError
		expect time.Duration
	}{
		{"credential invalid defaults to ten minutes", &CarrierError{Code: CodeCredentialInvalid}, 10 * time.Minute},
		{"rate limited defaults to a minute", &CarrierError{Code: CodeRateLimited}, time.Minute},
		{"unavailable has none by default", &CarrierError{Code: CodeCarrierUnavailable}, 0},
		{"busy has none", &CarrierError{Code: CodeServiceBusy}, 0},
		{"not found has none", &CarrierError{Code: CodeNotFound}, 0},
		{"an explicit value wins", &CarrierError{Code: CodeCarrierUnavailable, Cooldown: time.Minute}, time.Minute},
	}
	for _, tt := range tests {
		if got := tt.err.cooldown(); got != tt.expect {
			t.Errorf("%s: got %v, expected %v", tt.name, got, tt.expect)
		}
	}
}
