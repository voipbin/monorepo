package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// capturingHook records logrus entries for assertions without touching
// os.Stdout. It implements logrus.Hook.
type capturingHook struct {
	mu      sync.Mutex
	entries []*logrus.Entry
}

func (h *capturingHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *capturingHook) Fire(entry *logrus.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = append(h.entries, entry)
	return nil
}

func (h *capturingHook) waitForMessageContaining(t *testing.T, substr string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		h.mu.Lock()
		for _, e := range h.entries {
			if contains(e.Message, substr) {
				h.mu.Unlock()
				return
			}
		}
		h.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for a log message containing %q", substr)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (func() bool {
		for i := 0; i+len(substr) <= len(s); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	})()
}

// withCapturingHook installs a fresh hook on the standard logger for the
// duration of the test and restores the original hooks afterward, so this
// test's goroutine cannot leak log entries into a later test.
func withCapturingHook(t *testing.T) *capturingHook {
	t.Helper()

	stdLogger := logrus.StandardLogger()
	original := stdLogger.Hooks
	hook := &capturingHook{}
	stdLogger.AddHook(hook)
	t.Cleanup(func() {
		stdLogger.ReplaceHooks(original)
	})

	return hook
}

// Test_initProm_listenError verifies the ListenAndServe failure branch is
// reached and logged, using a listen address that net.Listen rejects
// immediately (missing port) so no real port is bound.
func Test_initProm_listenError(t *testing.T) {
	hook := withCapturingHook(t)

	endpoint := fmt.Sprintf("/metrics-test-error-%d", time.Now().UnixNano())
	initProm(endpoint, "bad-listen-address")

	hook.waitForMessageContaining(t, "Prometheus server error", 3*time.Second)

	_, pattern := http.DefaultServeMux.Handler(httptest.NewRequest(http.MethodGet, endpoint, nil))
	if pattern != endpoint {
		t.Errorf("Wrong match. expected: %s registered, got: %q", endpoint, pattern)
	}
}
