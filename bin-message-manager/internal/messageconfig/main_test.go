package messageconfig

import (
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/internal/configkeys"
	"monorepo/bin-message-manager/internal/logtest"
)

// levelOf returns the level of the first captured entry whose message contains the text, or false.
func levelOf(logs *logtest.Capture, text string) (logrus.Level, bool) {
	for _, e := range logs.Entries() {
		if strings.Contains(e.Message, text) {
			return e.Level, true
		}
	}

	return 0, false
}

// Test_tokens checks the environment values of the two tokens: set, trimmed, empty, unset and an
// unresolved placeholder. A token that is not configured is an empty value and no error, and the
// two tokens are independent. A store that fails when it is touched is alive next to the suppliers.
func Test_tokens(t *testing.T) {
	tests := []struct {
		name string

		messagebird string
		telnyx      string

		expectMessagebird string
		expectTelnyx      string
	}{
		{name: "unset"},
		{name: "both set", messagebird: "fake-mb", telnyx: "fake-tn", expectMessagebird: "fake-mb", expectTelnyx: "fake-tn"},
		{name: "telnyx only", telnyx: "fake-tn", expectTelnyx: "fake-tn"},
		{name: "messagebird only", messagebird: "fake-mb", expectMessagebird: "fake-mb"},
		{name: "spaces are trimmed", messagebird: "  fake-mb\n", telnyx: " fake-tn ", expectMessagebird: "fake-mb", expectTelnyx: "fake-tn"},
		{name: "empty and white space", messagebird: "   ", telnyx: ""},
		{name: "placeholder", messagebird: "[[BIN_MANAGER__MESSAGEBIRD_API_KEY]]", telnyx: "[[BIN_MANAGER__TELNYX_API_KEY]]"},
		{name: "placeholder is only a prefix rule", messagebird: "fake-[[mid", expectMessagebird: "fake-[[mid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(configkeys.EnvMessagebirdAPIKey, tt.messagebird)
			t.Setenv(configkeys.EnvTelnyxAPIKey, tt.telnyx)

			mb, err := MessagebirdToken()(context.Background())
			if err != nil || mb != tt.expectMessagebird {
				t.Errorf("Wrong match. messagebird expect: %q, got: %q %v", tt.expectMessagebird, mb, err)
			}
			tn, err := TelnyxToken()(context.Background())
			if err != nil || tn != tt.expectTelnyx {
				t.Errorf("Wrong match. telnyx expect: %q, got: %q %v", tt.expectTelnyx, tn, err)
			}
		})
	}
}

// The variables of the old releases are not read.
func Test_tokens_oldVariablesAreNotRead(t *testing.T) {
	t.Setenv("AUTHTOKEN_MESSAGEBIRD", "old-fake-mb")
	t.Setenv("AUTHTOKEN_TELNYX", "old-fake-tn")
	t.Setenv(configkeys.EnvMessagebirdAPIKey, "")
	t.Setenv(configkeys.EnvTelnyxAPIKey, "")

	if v, err := MessagebirdToken()(context.Background()); err != nil || v != "" {
		t.Errorf("Wrong match. messagebird expect: empty, got: %q %v", v, err)
	}
	if v, err := TelnyxToken()(context.Background()); err != nil || v != "" {
		t.Errorf("Wrong match. telnyx expect: empty, got: %q %v", v, err)
	}
}

// The suppliers keep nothing: a changed variable is the next call's result.
func Test_tokens_nextCallSeesTheChange(t *testing.T) {
	supplier := TelnyxToken()
	ctx := context.Background()

	t.Setenv(configkeys.EnvTelnyxAPIKey, "")
	if got, err := supplier(ctx); err != nil || got != "" {
		t.Fatalf("Wrong match. expect: empty, got: %q %v", got, err)
	}
	t.Setenv(configkeys.EnvTelnyxAPIKey, "token-1")
	if got, _ := supplier(ctx); got != "token-1" {
		t.Errorf("Wrong match. expect: token-1, got: %q", got)
	}
	t.Setenv(configkeys.EnvTelnyxAPIKey, "token-2")
	if got, _ := supplier(ctx); got != "token-2" {
		t.Errorf("Wrong match. expect: token-2, got: %q", got)
	}
}

// LogStates: one line per variable. A variable that is not set is a warning for Telnyx (the first
// provider of a send from a Telnyx number or from a number without an owner) and an info line for the others; a
// placeholder is a warning with the words "unresolved placeholder"; a set variable is an info line; a callback URL
// that is set but not valid is a warning; no line holds a value.
func Test_LogStates(t *testing.T) {
	const (
		unsetText       = "The variable is not set. variable: "
		placeholderText = "The variable holds an unresolved placeholder. variable: "
		setText         = "The variable is set. variable: "
		invalidText     = "The variable is set but it is not a valid callback URL, so it is ignored. variable: "
	)

	type state struct {
		level logrus.Level
		text  string
	}

	names := []string{
		configkeys.EnvMessagebirdAPIKey,
		configkeys.EnvTelnyxAPIKey,
		configkeys.EnvTwilioAccountSID,
		configkeys.EnvTwilioAuthToken,
		configkeys.EnvTwilioStatusCallbackURL,
	}

	allUnset := []state{{logrus.InfoLevel, unsetText}, {logrus.WarnLevel, unsetText}, {logrus.InfoLevel, unsetText}, {logrus.InfoLevel, unsetText}, {logrus.InfoLevel, unsetText}}

	tests := []struct {
		name string

		values []string // in the order of names

		expect []state
	}{
		{name: "unset", values: []string{"", "", "", "", ""}, expect: allUnset},
		{name: "empty", values: []string{"  ", "", " ", "", "\t"}, expect: allUnset},
		{
			name:   "placeholder",
			values: []string{"[[X]]", "[[Y]]", "[[Z]]", "[[W]]", "[[V]]"},
			expect: []state{{logrus.WarnLevel, placeholderText}, {logrus.WarnLevel, placeholderText}, {logrus.WarnLevel, placeholderText}, {logrus.WarnLevel, placeholderText}, {logrus.WarnLevel, placeholderText}},
		},
		{
			name:   "set",
			values: []string{"log-fake-mb", "log-fake-tn", "log-fake-sid", "log-fake-token", "https://hook.example.com/v1.0/messages/twilio"},
			expect: []state{{logrus.InfoLevel, setText}, {logrus.InfoLevel, setText}, {logrus.InfoLevel, setText}, {logrus.InfoLevel, setText}, {logrus.InfoLevel, setText}},
		},
		{
			name:   "a callback url that is not valid",
			values: []string{"", "", "", "", "https://hook.example.com/other"},
			expect: []state{{logrus.InfoLevel, unsetText}, {logrus.WarnLevel, unsetText}, {logrus.InfoLevel, unsetText}, {logrus.InfoLevel, unsetText}, {logrus.WarnLevel, invalidText}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, name := range names {
				t.Setenv(name, tt.values[i])
			}

			logs := logtest.New(t)
			LogStates()

			if logs.Len() != len(names) {
				t.Errorf("Wrong match. expect: %d lines, got: %s", len(names), logs.Dump())
			}
			for i, name := range names {
				if lv, ok := levelOf(logs, tt.expect[i].text+name); !ok || lv != tt.expect[i].level {
					t.Errorf("Wrong match. %s expect: %v, got: %v %v, %s", name, tt.expect[i].level, lv, ok, logs.Dump())
				}
			}
			logs.AssertAbsent(t, "log-fake", "[[X]]", "[[Y]]", "[[Z]]", "[[W]]", "[[V]]", "hook.example.com")
		})
	}
}

// The Twilio credentials are a pair: half of it is the same as nothing, and a placeholder is not a value.
func Test_TwilioCredentials(t *testing.T) {
	tests := []struct {
		name string

		sid   string
		token string

		expectSID   string
		expectToken string
	}{
		{"both set", " AC-fake-sid ", " fake-token\n", "AC-fake-sid", "fake-token"},
		{"no sid", "", "fake-token", "", ""},
		{"no token", "AC-fake-sid", "  ", "", ""},
		{"neither", "", "", "", ""},
		{"placeholders", "[[BIN_MANAGER__TWILIO_ACCOUNT_SID]]", "[[BIN_MANAGER__TWILIO_AUTH_TOKEN]]", "", ""},
		{"a placeholder token", "AC-fake-sid", "[[BIN_MANAGER__TWILIO_AUTH_TOKEN]]", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(configkeys.EnvTwilioAccountSID, tt.sid)
			t.Setenv(configkeys.EnvTwilioAuthToken, tt.token)

			sid, token, err := TwilioCredentials()(context.Background())
			if err != nil || sid != tt.expectSID || token != tt.expectToken {
				t.Errorf("Wrong match. expect: %q %q, got: %q %q %v", tt.expectSID, tt.expectToken, sid, token, err)
			}
		})
	}
}

// The callback URL is valid only as a plain http or https URL of the hook route; the supplier hands out nothing else.
func Test_ValidCallbackURL(t *testing.T) {
	tests := []struct {
		name  string
		value string

		expect bool
	}{
		{"https", "https://hook.example.com/v1.0/messages/twilio", true},
		{"http", "http://hook.example.com/v1.0/messages/twilio", true},
		{"a port that is not the default", "https://hook.example.com:8443/v1.0/messages/twilio", true},
		{"unset", "", false},
		{"explicit default https port", "https://hook.example.com:443/v1.0/messages/twilio", false},
		{"explicit default http port", "http://hook.example.com:80/v1.0/messages/twilio", false},
		{"user information", "https://user:pass@hook.example.com/v1.0/messages/twilio", false},
		{"query", "https://hook.example.com/v1.0/messages/twilio?a=1", false},
		{"empty query mark", "https://hook.example.com/v1.0/messages/twilio?", false},
		{"fragment", "https://hook.example.com/v1.0/messages/twilio#x", false},
		{"wrong path", "https://hook.example.com/v1.0/messages/telnyx", false},
		{"no path", "https://hook.example.com", false},
		{"no host", "https:///v1.0/messages/twilio", false},
		{"not a url", "hook.example.com/v1.0/messages/twilio", false},
		{"another scheme", "ftp://hook.example.com/v1.0/messages/twilio", false},
		{"placeholder", "[[BIN_MANAGER__TWILIO_STATUS_CALLBACK_URL]]", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidCallbackURL(tt.value); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}

			t.Setenv(configkeys.EnvTwilioStatusCallbackURL, tt.value)
			want := ""
			if tt.expect {
				want = tt.value
			}
			if got := TwilioStatusCallbackURL()(context.Background()); got != want {
				t.Errorf("Wrong match. supplier expect: %q, got: %q", want, got)
			}
		})
	}
}
