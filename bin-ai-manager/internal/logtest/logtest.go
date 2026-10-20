// Package logtest captures the standard logrus logger in tests, so that a test can prove that a
// code path does not write a given text (for example a conversation sentence) into the logs.
package logtest

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

// Capture holds the entries written to the standard logger while it is active.
type Capture struct {
	hook *logrustest.Hook
}

// New attaches a hook to the standard logger and sets the Debug level, so that the Debug lines are
// collected too. The level, the output and the hooks are restored when the test ends.
func New(t *testing.T) *Capture {
	t.Helper()

	hook := logrustest.NewLocal(logrus.StandardLogger())
	prev := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() {
		logrus.SetLevel(prev)
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	})

	return &Capture{hook: hook}
}

// Dump renders every entry the way a reader of the log sees it: the message, the fields as JSON (the
// production formatter marshals fields) and the fields printed with %v.
func (c *Capture) Dump() string {
	var sb strings.Builder
	for _, e := range c.hook.AllEntries() {
		fields := make(logrus.Fields, len(e.Data))
		for k, v := range e.Data {
			// a []byte is a text for the reader, but json prints base64 and %v prints numbers
			if b, ok := v.([]byte); ok {
				v = string(b)
			}
			fields[k] = v
		}
		e.Data = fields
		sb.WriteString(e.Message)
		sb.WriteString("\n")
		if b, err := json.Marshal(e.Data); err == nil {
			sb.Write(b)
		} else {
			sb.WriteString("(fields not marshalable)")
		}
		sb.WriteString("\n")
		fmt.Fprintf(&sb, "%v", e.Data)
		sb.WriteString("\n")
	}
	return sb.String()
}

// Len returns the number of captured entries.
func (c *Capture) Len() int {
	return len(c.hook.AllEntries())
}

// HasField reports whether any captured entry has the field key.
func (c *Capture) HasField(key string) bool {
	for _, e := range c.hook.AllEntries() {
		if _, ok := e.Data[key]; ok {
			return true
		}
	}
	return false
}

// AssertAbsent fails when any captured text contains one of the texts. It also fails when nothing was
// captured, because an empty capture would make the assertion pass vacuously.
func (c *Capture) AssertAbsent(t *testing.T, texts ...string) {
	t.Helper()

	if c.Len() == 0 {
		t.Fatalf("no log entry was captured, the assertion would be vacuous")
	}

	dump := c.Dump()
	for _, text := range texts {
		// the base64 form is checked too, as a []byte inside a nested value is printed that way
		if strings.Contains(dump, text) || strings.Contains(dump, base64.StdEncoding.EncodeToString([]byte(text))) {
			t.Errorf("the text must not be logged: %q", text)
		}
	}
}

// AssertField fails when no captured entry has the field key.
func (c *Capture) AssertField(t *testing.T, key string) {
	t.Helper()

	if !c.HasField(key) {
		t.Errorf("the log must keep the field %q", key)
	}
}
