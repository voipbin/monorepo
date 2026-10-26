package main

import (
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// capturedEntry is a copy of one log entry, safe to read from any goroutine.
type capturedEntry struct {
	message string
	data    logrus.Fields
}

// logCollector is a logrus hook that keeps a copy of every entry it sees.
type logCollector struct {
	mu      sync.Mutex
	entries []capturedEntry
}

func (c *logCollector) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (c *logCollector) Fire(e *logrus.Entry) error {
	data := logrus.Fields{}
	for k, v := range e.Data {
		data[k] = v
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = append(c.entries, capturedEntry{message: e.Message, data: data})

	return nil
}

// captureLogs installs a collecting hook on the standard logger. The hook set is
// swapped for a new empty one first, because AddHook modifies the live map, and
// the original set is restored when the test ends.
func captureLogs(t *testing.T) *logCollector {
	t.Helper()

	c := &logCollector{}
	saved := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	logrus.AddHook(c)
	t.Cleanup(func() {
		logrus.StandardLogger().ReplaceHooks(saved)
	})

	return c
}

// waitFor polls until an entry matches, or fails the test after five seconds.
func (c *logCollector) waitFor(t *testing.T, desc string, match func(e capturedEntry) bool) capturedEntry {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		c.mu.Lock()
		entries := append([]capturedEntry(nil), c.entries...)
		c.mu.Unlock()

		for _, e := range entries {
			if match(e) {
				return e
			}
		}

		if time.Now().After(deadline) {
			t.Fatalf("no log entry matched in time. expect: %s", desc)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
