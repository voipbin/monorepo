package billingconfig

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/internal/configkeys"
	"monorepo/bin-billing-manager/internal/logtest"
)

// The suppliers read the variable of the key at the time of use, trim it, and treat an unset,
// empty or unresolved placeholder value as not configured (an empty value, no error).
func Test_Supplier(t *testing.T) {
	tests := []struct {
		name string

		value string
		set   bool

		expect string
	}{
		{"unset", "", false, ""},
		{"empty", "", true, ""},
		{"spaces only", "  \t", true, ""},
		{"value", "fake-value", true, "fake-value"},
		{"value with surrounding space", " fake-value\n", true, "fake-value"},
		{"placeholder", "[[BIN_MANAGER__X]]", true, ""},
	}

	for _, key := range []string{configkeys.EnvPaddleAPIKey, configkeys.EnvPaddleWebhookSecret} {
		for _, tt := range tests {
			t.Run(key+" "+tt.name, func(t *testing.T) {
				if tt.set {
					t.Setenv(key, tt.value)
				} else {
					t.Setenv(key, "")
					_ = os.Unsetenv(key)
				}

				got, err := Supplier(key)(context.Background())
				if err != nil || got != tt.expect {
					t.Errorf("Wrong match. expect: %q, got: %q %v", tt.expect, got, err)
				}
			})
		}
	}
}

// The supplier keeps nothing: a changed variable is used by the next call.
func Test_Supplier_resolvedAtRequestTime(t *testing.T) {
	supplier := Supplier(configkeys.EnvPaddleWebhookSecret)

	for _, v := range []string{"fake-1", "fake-2", "", "fake-3"} {
		t.Setenv(configkeys.EnvPaddleWebhookSecret, v)
		if got, err := supplier(context.Background()); err != nil || got != v {
			t.Errorf("Wrong match. expect: %q, got: %q %v", v, got, err)
		}
	}
}

// LogStates: the webhook secret that is not set is a warning (every webhook is rejected), the API
// key that is not set is an info line, a set variable is an info line, a placeholder is a warning
// that names the variable, and no line holds a value.
func Test_LogStates(t *testing.T) {
	levelOf := func(logs *logtest.Capture, name string) (logrus.Level, string) {
		for _, e := range logs.Entries() {
			if e.Data["variable"] == name {
				return e.Level, e.Message
			}
		}
		return logrus.PanicLevel, ""
	}

	t.Run("nothing set", func(t *testing.T) {
		t.Setenv(configkeys.EnvPaddleAPIKey, "")
		t.Setenv(configkeys.EnvPaddleWebhookSecret, "")
		logs := logtest.New(t)
		LogStates()

		if lvl, msg := levelOf(logs, configkeys.EnvPaddleWebhookSecret); lvl != logrus.WarnLevel || !strings.Contains(msg, "every Paddle webhook is rejected") {
			t.Errorf("Wrong match. expect: a warning that webhooks are rejected, got: %v %s", lvl, msg)
		}
		if lvl, _ := levelOf(logs, configkeys.EnvPaddleAPIKey); lvl != logrus.InfoLevel {
			t.Errorf("Wrong match. expect: an info line for the API key, got: %v", lvl)
		}
		if logs.HasLevel(logrus.ErrorLevel) {
			t.Errorf("Wrong match. expect: no error, got: %s", logs.Dump())
		}
	})

	t.Run("set", func(t *testing.T) {
		t.Setenv(configkeys.EnvPaddleAPIKey, "fake-api-text")
		t.Setenv(configkeys.EnvPaddleWebhookSecret, "fake-webhook-text")
		logs := logtest.New(t)
		LogStates()

		for _, e := range logs.Entries() {
			if e.Level != logrus.InfoLevel {
				t.Errorf("Wrong match. expect: info lines only, got: %v %s", e.Level, e.Message)
			}
		}
		if logs.Len() != 2 {
			t.Errorf("Wrong match. expect: one line per variable, got: %s", logs.Dump())
		}
		logs.AssertAbsent(t, "fake-api-text", "fake-webhook-text")
	})

	t.Run("placeholder", func(t *testing.T) {
		t.Setenv(configkeys.EnvPaddleAPIKey, "[[BIN_MANAGER__PADDLE_API_KEY]]")
		t.Setenv(configkeys.EnvPaddleWebhookSecret, "[[BIN_MANAGER__PADDLE_WEBHOOK_SECRET]]")
		logs := logtest.New(t)
		LogStates()

		for _, name := range []string{configkeys.EnvPaddleAPIKey, configkeys.EnvPaddleWebhookSecret} {
			lvl, msg := levelOf(logs, name)
			if lvl != logrus.WarnLevel || !strings.Contains(msg, "unresolved placeholder") || !strings.Contains(msg, name) {
				t.Errorf("Wrong match. %s: expect: a warning naming the variable, got: %v %s", name, lvl, msg)
			}
		}
		logs.AssertAbsent(t, "BIN_MANAGER__PADDLE")
	})
}
