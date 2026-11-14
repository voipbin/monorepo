package transcribeconfig

import (
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/internal/configkeys"
	"monorepo/bin-transcribe-manager/internal/logtest"
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

// Test_AWSKeys checks the environment pair: both set is the pair (trimmed), neither or a
// placeholder is not configured (an empty pair, no error), exactly one is an empty pair and no error.
// A store that fails when it is touched lives next to the supplier.
func Test_AWSKeys(t *testing.T) {
	tests := []struct {
		name string

		access string
		secret string

		expectAccess string
		expectSecret string
	}{
		{name: "unset"},
		{name: "both set", access: "fake-access", secret: "fake-secret", expectAccess: "fake-access", expectSecret: "fake-secret"},
		{name: "spaces are trimmed", access: " fake-access\n", secret: "  fake-secret ", expectAccess: "fake-access", expectSecret: "fake-secret"},
		{name: "both empty", access: "  ", secret: ""},
		{name: "both placeholder", access: "[[BIN_MANAGER__AWS_ACCESS_KEY]]", secret: "[[BIN_MANAGER__AWS_SECRET_KEY]]"},
		{name: "access only", access: "fake-access"},
		{name: "secret only", secret: "fake-secret"},
		{name: "access set and secret placeholder", access: "fake-access", secret: "[[BIN_MANAGER__AWS_SECRET_KEY]]"},
		{name: "access placeholder and secret set", access: "[[BIN_MANAGER__AWS_ACCESS_KEY]]", secret: "fake-secret"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(configkeys.EnvAWSAccessKey, tt.access)
			t.Setenv(configkeys.EnvAWSSecretKey, tt.secret)

			a, s, err := AWSKeys()(context.Background())
			if err != nil || a != tt.expectAccess || s != tt.expectSecret {
				t.Errorf("Wrong match. expect: %q %q, got: %q %q %v", tt.expectAccess, tt.expectSecret, a, s, err)
			}
		})
	}
}

// A half pair is logged once at start, never per call.
func Test_AWSKeys_logsNothingPerCall(t *testing.T) {
	t.Setenv(configkeys.EnvAWSAccessKey, "fake-access")
	t.Setenv(configkeys.EnvAWSSecretKey, "")

	logs := logtest.New(t)
	for i := 0; i < 3; i++ {
		if _, _, err := AWSKeys()(context.Background()); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}
	if logs.Len() != 0 {
		t.Errorf("Wrong match. expect: no log line, got: %s", logs.Dump())
	}
}

// LogStates: one line per variable (info when not set, because AWS is optional; warn for a
// placeholder; info when set), and one more warning when exactly one of the two is set. No line
// holds a value.
func Test_LogStates(t *testing.T) {
	const (
		unsetText       = "The variable is not set. variable: "
		placeholderText = "The variable holds an unresolved placeholder. variable: "
		setText         = "The variable is set. variable: "
		halfText        = "Only one of the two AWS variables is set, so AWS is not available."
	)

	tests := []struct {
		name string

		access string
		secret string

		expectAccess logrus.Level
		expectSecret logrus.Level
		expectTexts  [2]string
		expectHalf   bool
	}{
		{name: "unset", expectAccess: logrus.InfoLevel, expectSecret: logrus.InfoLevel, expectTexts: [2]string{unsetText, unsetText}},
		{name: "both set", access: "log-fake-a", secret: "log-fake-s", expectAccess: logrus.InfoLevel, expectSecret: logrus.InfoLevel, expectTexts: [2]string{setText, setText}},
		{name: "placeholders", access: "[[A]]", secret: "[[S]]", expectAccess: logrus.WarnLevel, expectSecret: logrus.WarnLevel, expectTexts: [2]string{placeholderText, placeholderText}},
		{name: "access only", access: "log-fake-a", expectAccess: logrus.InfoLevel, expectSecret: logrus.InfoLevel, expectTexts: [2]string{setText, unsetText}, expectHalf: true},
		{name: "secret only", secret: "log-fake-s", expectAccess: logrus.InfoLevel, expectSecret: logrus.InfoLevel, expectTexts: [2]string{unsetText, setText}, expectHalf: true},
		{name: "access set and secret placeholder", access: "log-fake-a", secret: "[[S]]", expectAccess: logrus.InfoLevel, expectSecret: logrus.WarnLevel, expectTexts: [2]string{setText, placeholderText}, expectHalf: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(configkeys.EnvAWSAccessKey, tt.access)
			t.Setenv(configkeys.EnvAWSSecretKey, tt.secret)

			logs := logtest.New(t)
			LogStates()

			expectLines := 2
			if tt.expectHalf {
				expectLines = 3
			}
			if logs.Len() != expectLines {
				t.Errorf("Wrong match. expect: %d lines, got: %s", expectLines, logs.Dump())
			}
			if lv, ok := levelOf(logs, tt.expectTexts[0]+configkeys.EnvAWSAccessKey); !ok || lv != tt.expectAccess {
				t.Errorf("Wrong match. access expect: %v, got: %v %v, %s", tt.expectAccess, lv, ok, logs.Dump())
			}
			if lv, ok := levelOf(logs, tt.expectTexts[1]+configkeys.EnvAWSSecretKey); !ok || lv != tt.expectSecret {
				t.Errorf("Wrong match. secret expect: %v, got: %v %v, %s", tt.expectSecret, lv, ok, logs.Dump())
			}
			lv, ok := levelOf(logs, halfText)
			if ok != tt.expectHalf || (ok && lv != logrus.WarnLevel) {
				t.Errorf("Wrong match. half pair expect: %v warn, got: %v %v", tt.expectHalf, lv, ok)
			}
			logs.AssertAbsent(t, "log-fake", "[[A]]", "[[S]]")
		})
	}
}
