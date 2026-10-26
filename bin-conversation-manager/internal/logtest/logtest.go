// Package logtest helps tests assert that a secret never reaches the service log.
package logtest

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

// forbiddenFieldTypes are the wire and account types that carry credentials or raw request bodies.
// A log field of one of these types leaks its content in the text format, whatever the sentinel looks like.
var forbiddenFieldTypes = map[string]bool{
	"*sock.Request":                  true,
	"sock.Request":                   true,
	"*account.Account":               true,
	"[]*account.Account":             true,
	"map[account.Field]any":          true,
	"map[account.Field]interface {}": true,
	"request.V1DataHooksPost":        true,
	"*request.V1DataHooksPost":       true,
}

// Capture installs a global logrus hook at debug level and restores the logger when the test ends.
// The logger is global, so tests that use it must not run in parallel.
func Capture(t *testing.T) *test.Hook {
	t.Helper()

	std := logrus.StandardLogger()
	prevLevel := std.GetLevel()
	prevOut := std.Out
	prevHooks := std.ReplaceHooks(make(logrus.LevelHooks))

	hook := test.NewGlobal()
	std.SetLevel(logrus.DebugLevel)
	std.SetOutput(io.Discard)

	t.Cleanup(func() {
		std.ReplaceHooks(prevHooks)
		std.SetLevel(prevLevel)
		std.SetOutput(prevOut)
	})

	return hook
}

// variants returns the forms in which a sentinel can show up in a log entry:
// plain, base64 (the JSON formatter encodes []byte), the decimal list of a []byte printed with %v, and URL escaped.
func variants(sentinel string) []string {
	raw := []byte(sentinel)

	return []string{
		sentinel,
		base64.StdEncoding.EncodeToString(raw),
		fmt.Sprintf("%d", raw),
		url.QueryEscape(sentinel),
	}
}

func containsSentinel(renderings []string, sentinels []string) bool {
	for _, r := range renderings {
		for _, s := range sentinels {
			for _, v := range variants(s) {
				if strings.Contains(r, v) {
					return true
				}
			}
		}
	}

	return false
}

// AssertNoLeak fails when any recorded entry contains one of the sentinels in any rendering,
// or carries a field value of a forbidden type. At least one entry must exist, otherwise
// the assertion would pass without having observed anything.
func AssertNoLeak(t *testing.T, hook *test.Hook, sentinels ...string) {
	t.Helper()

	entries := hook.AllEntries()
	if len(entries) == 0 {
		t.Errorf("no log entry was recorded, the assertion would be vacuous")
		return
	}

	jsonFormatter := &logrus.JSONFormatter{}
	textFormatter := &logrus.TextFormatter{DisableColors: true, DisableTimestamp: true}

	for _, e := range entries {
		renderings := []string{e.Message}
		if b, err := jsonFormatter.Format(e); err == nil {
			renderings = append(renderings, string(b))
		}
		if b, err := textFormatter.Format(e); err == nil {
			renderings = append(renderings, string(b))
		}

		for k, v := range e.Data {
			if forbiddenFieldTypes[fmt.Sprintf("%T", v)] {
				t.Errorf("log field %q has forbidden type %T. message: %q", k, v, e.Message)
			}
			renderings = append(renderings, fmt.Sprint(v), fmt.Sprintf("%+v", v), fmt.Sprintf("%#v", v))
		}

		if containsSentinel(renderings, sentinels) {
			t.Errorf("the secret leaked to the log. message: %q", e.Message)
		}
	}
}
