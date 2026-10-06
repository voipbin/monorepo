package servicehandler

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

const (
	secretLogPassword   = "sentinel-password-must-not-leak-4d2c"
	secretLogDirectHash = "sentinel-direct-hash-must-not-leak-8b1f"
)

// newSecretLogHook attaches a hook to the standard logger and enables the debug
// level, because the log calls under test are Debug and would otherwise be
// dropped, which would make the assertion pass vacuously.
func newSecretLogHook(t *testing.T) *logrustest.Hook {
	t.Helper()

	hook := logrustest.NewLocal(logrus.StandardLogger())
	prev := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() {
		logrus.SetLevel(prev)
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	})
	return hook
}

// assertNoSecretInLogs requires at least one collected entry whose message
// contains wantMsg, and that no field of any entry carries a secret, both when
// the field is JSON-marshaled (the production formatter) and when it is printed
// with %v.
func assertNoSecretInLogs(t *testing.T, hook *logrustest.Hook, wantMsg string, secrets ...string) {
	t.Helper()

	found := 0
	for _, entry := range hook.AllEntries() {
		if strings.Contains(entry.Message, wantMsg) {
			found++
		}
		b, err := json.Marshal(entry.Data)
		if err != nil {
			t.Fatalf("could not marshal the log fields: %v", err)
		}
		for _, secret := range secrets {
			if strings.Contains(entry.Message, secret) {
				t.Errorf("secret leaked into the log message: %q", entry.Message)
			}
			if strings.Contains(string(b), secret) {
				t.Errorf("secret leaked into the JSON log fields of %q: %s", entry.Message, b)
			}
			if strings.Contains(fmt.Sprintf("%v", entry.Data), secret) {
				t.Errorf("secret leaked into the %%v log fields of %q", entry.Message)
			}
		}
	}
	if found == 0 {
		t.Fatalf("no log entry with message %q was collected, the assertion would be vacuous", wantMsg)
	}
}
