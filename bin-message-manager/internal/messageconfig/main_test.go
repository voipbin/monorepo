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
// provider of every send) and an info line for MessageBird; a placeholder is a warning with the
// words "unresolved placeholder"; a set variable is an info line; no line holds a value.
func Test_LogStates(t *testing.T) {
	const (
		unsetText       = "The variable is not set. variable: "
		placeholderText = "The variable holds an unresolved placeholder. variable: "
		setText         = "The variable is set. variable: "
	)

	tests := []struct {
		name string

		messagebird string
		telnyx      string

		expectMessagebird logrus.Level
		expectTelnyx      logrus.Level
		expectText        [2]string
	}{
		{name: "unset", expectMessagebird: logrus.InfoLevel, expectTelnyx: logrus.WarnLevel, expectText: [2]string{unsetText, unsetText}},
		{name: "empty", messagebird: "  ", telnyx: "", expectMessagebird: logrus.InfoLevel, expectTelnyx: logrus.WarnLevel, expectText: [2]string{unsetText, unsetText}},
		{name: "placeholder", messagebird: "[[X]]", telnyx: "[[Y]]", expectMessagebird: logrus.WarnLevel, expectTelnyx: logrus.WarnLevel, expectText: [2]string{placeholderText, placeholderText}},
		{name: "set", messagebird: "log-fake-mb", telnyx: "log-fake-tn", expectMessagebird: logrus.InfoLevel, expectTelnyx: logrus.InfoLevel, expectText: [2]string{setText, setText}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(configkeys.EnvMessagebirdAPIKey, tt.messagebird)
			t.Setenv(configkeys.EnvTelnyxAPIKey, tt.telnyx)

			logs := logtest.New(t)
			LogStates()

			if logs.Len() != 2 {
				t.Errorf("Wrong match. expect: two lines, got: %s", logs.Dump())
			}
			if lv, ok := levelOf(logs, tt.expectText[0]+configkeys.EnvMessagebirdAPIKey); !ok || lv != tt.expectMessagebird {
				t.Errorf("Wrong match. messagebird expect: %v, got: %v %v, %s", tt.expectMessagebird, lv, ok, logs.Dump())
			}
			if lv, ok := levelOf(logs, tt.expectText[1]+configkeys.EnvTelnyxAPIKey); !ok || lv != tt.expectTelnyx {
				t.Errorf("Wrong match. telnyx expect: %v, got: %v %v, %s", tt.expectTelnyx, lv, ok, logs.Dump())
			}
			logs.AssertAbsent(t, "log-fake", "[[X]]", "[[Y]]")
		})
	}
}
