package emailconfig

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"monorepo/bin-email-manager/internal/configkeys"
)

// The suppliers read the variable at the time of use, trim it, and treat an unset, empty or
// unresolved placeholder value as not configured (an empty value, no error).
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

	for _, key := range []string{configkeys.EnvSendgridAPIKey, configkeys.EnvMailgunAPIKey, configkeys.EnvMailgunWebhookSigningKey} {
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

// Nothing is cached: a changed variable is used by the next call.
func Test_Supplier_resolvedAtRequestTime(t *testing.T) {
	supplier := Supplier(configkeys.EnvMailgunAPIKey)

	for _, v := range []string{"fake-1", "fake-2", "", "fake-3"} {
		t.Setenv(configkeys.EnvMailgunAPIKey, v)
		if got, err := supplier(context.Background()); err != nil || got != v {
			t.Errorf("Wrong match. expect: %q, got: %q %v", v, got, err)
		}
	}
}

// LogStates: every variable that is not set is a warning, a set variable is an info line, a
// placeholder is a warning that names the variable, and no line holds a value.
func Test_LogStates(t *testing.T) {
	names := []string{configkeys.EnvSendgridAPIKey, configkeys.EnvMailgunAPIKey, configkeys.EnvMailgunWebhookSigningKey}

	run := func(t *testing.T, value string) map[string]*logrus.Entry {
		t.Helper()

		for _, n := range names {
			t.Setenv(n, value)
		}
		logger, hook := logrustest.NewNullLogger()
		_ = logger
		logrus.AddHook(hook)
		t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks)) })

		LogStates()

		res := map[string]*logrus.Entry{}
		for _, e := range hook.AllEntries() {
			if v, ok := e.Data["variable"].(string); ok {
				res[v] = e
			}
			if strings.Contains(e.Message, "fake-secret-text") || strings.Contains(fmt.Sprint(e.Data), "fake-secret-text") {
				t.Errorf("Wrong match. a value was logged: %s", e.Message)
			}
		}
		if len(res) != len(names) {
			t.Errorf("Wrong match. expect: one line per variable, got: %d", len(res))
		}

		return res
	}

	t.Run("not set", func(t *testing.T) {
		for n, e := range run(t, "") {
			if e.Level != logrus.WarnLevel || !strings.Contains(e.Message, "not set") {
				t.Errorf("Wrong match. %s expect: a warning, got: %v %s", n, e.Level, e.Message)
			}
		}
	})

	t.Run("set", func(t *testing.T) {
		for n, e := range run(t, "fake-secret-text") {
			if e.Level != logrus.InfoLevel {
				t.Errorf("Wrong match. %s expect: info, got: %v %s", n, e.Level, e.Message)
			}
		}
	})

	t.Run("placeholder", func(t *testing.T) {
		for n, e := range run(t, "[[BIN_MANAGER__X]]") {
			if e.Level != logrus.WarnLevel || !strings.Contains(e.Message, "unresolved placeholder") || !strings.Contains(e.Message, n) {
				t.Errorf("Wrong match. %s expect: a warning naming the variable, got: %v %s", n, e.Level, e.Message)
			}
			if strings.Contains(e.Message, "BIN_MANAGER__X") {
				t.Errorf("Wrong match. the placeholder text was logged: %s", e.Message)
			}
		}
	})
}

func Test_Suppliers(t *testing.T) {
	t.Setenv(configkeys.EnvSendgridAPIKey, "sendgrid-value")
	t.Setenv(configkeys.EnvMailgunAPIKey, "mailgun-value")
	t.Setenv(configkeys.EnvMailgunWebhookSigningKey, "signing-value")

	sendgrid, mailgun, signing := Suppliers()

	tests := []struct {
		name   string
		f      func(ctx context.Context) (string, error)
		expect string
	}{
		{"sendgrid is the first", sendgrid, "sendgrid-value"},
		{"mailgun is the second", mailgun, "mailgun-value"},
		{"signing key is the third", signing, "signing-value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.f(context.Background())
			if err != nil || got != tt.expect {
				t.Errorf("Wrong match. expect: (%q, nil), got: (%q, %v)", tt.expect, got, err)
			}
		})
	}
}
