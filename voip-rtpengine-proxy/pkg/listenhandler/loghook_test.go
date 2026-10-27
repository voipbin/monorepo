package listenhandler

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// capturedEntry is a copy of a log entry taken inside the hook, so tests can
// read it while goroutines keep logging.
type capturedEntry struct {
	level   logrus.Level
	message string
	fields  logrus.Fields
}

// logCapture is a logrus hook that records every entry of the standard logger.
type logCapture struct {
	mu      sync.Mutex
	entries []capturedEntry
}

func (c *logCapture) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (c *logCapture) Fire(e *logrus.Entry) error {
	fields := logrus.Fields{}
	for k, v := range e.Data {
		fields[k] = v
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = append(c.entries, capturedEntry{level: e.Level, message: e.Message, fields: fields})
	return nil
}

// captureLogs replaces the hooks of the standard logger with a fresh capture
// and restores the previous hooks when the test ends.
func captureLogs(t *testing.T) *logCapture {
	t.Helper()

	c := &logCapture{}
	hooks := make(logrus.LevelHooks)
	hooks.Add(c)
	previous := logrus.StandardLogger().ReplaceHooks(hooks)
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(previous) })
	return c
}

func (c *logCapture) count(match func(capturedEntry) bool) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	n := 0
	for _, e := range c.entries {
		if match(e) {
			n++
		}
	}
	return n
}

// waitFor polls until an entry matches or the deadline passes.
func (c *logCapture) waitFor(t *testing.T, what string, match func(capturedEntry) bool) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if c.count(match) > 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for log: %s", what)
}

// msgIs matches the exact message and every given field value.
func msgIs(level logrus.Level, message string, fields ...interface{}) func(capturedEntry) bool {
	return func(e capturedEntry) bool {
		if e.level != level || e.message != message {
			return false
		}
		for i := 0; i+1 < len(fields); i += 2 {
			if fmt.Sprint(e.fields[fields[i].(string)]) != fmt.Sprint(fields[i+1]) {
				return false
			}
		}
		return true
	}
}
