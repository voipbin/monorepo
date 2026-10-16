package rabbitmqhandler

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// captureHook collects the log entries as formatted text (message and every field).
type captureHook struct {
	mu      sync.Mutex
	entries []string
}

func (h *captureHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *captureHook) Fire(e *logrus.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = append(h.entries, fmt.Sprintf("%s %v", e.Message, e.Data))

	return nil
}

func (h *captureHook) snapshot() []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	return append([]string(nil), h.entries...)
}

// waitFor waits until an entry containing all of the given parts exists.
func (h *captureHook) waitFor(t *testing.T, parts ...string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, e := range h.snapshot() {
			ok := true
			for _, p := range parts {
				if !strings.Contains(e, p) {
					ok = false
					break
				}
			}
			if ok {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no log entry with %v. entries: %v", parts, h.snapshot())
}

func installCaptureHook(t *testing.T) *captureHook {
	t.Helper()

	std := logrus.StandardLogger()
	oldLevel := std.GetLevel()
	oldHooks := std.ReplaceHooks(make(logrus.LevelHooks))
	h := &captureHook{}
	std.AddHook(h)
	std.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() {
		std.SetLevel(oldLevel)
		std.ReplaceHooks(oldHooks)
	})

	return h
}

func Test_uriHost(t *testing.T) {
	tests := []struct {
		name string
		uri  string

		expectHost  string
		expectValid bool
	}{
		{"normal", "amqp://user:SENTINEL@broker.example:5672/", "broker.example:5672", true},
		{"without credentials", "amqp://broker.example", "broker.example", true},
		{"bad port", "amqp://user:SENTINEL@broker.example:99x/", "", false},
		{"bad escape in path", "amqp://user:SENTINEL@broker.example/%zz", "", false},
		{"bad escape in password", "amqp://user:SENTINEL%zz@broker.example/", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, valid := uriHost(tt.uri)
			if host != tt.expectHost || valid != tt.expectValid {
				t.Errorf("Wrong match. expect: (%q, %v), got: (%q, %v)", tt.expectHost, tt.expectValid, host, valid)
			}
			if strings.Contains(host, "SENTINEL") {
				t.Errorf("host contains the credential: %q", host)
			}
		})
	}
}

func Test_Close_no_credentials(t *testing.T) {
	h := installCaptureHook(t)

	r := &rabbit{
		uri:        "amqp://user:SENTINEL@broker.example:5672/",
		connection: newMockConnection(),
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string][]*queueBind),
	}
	r.Close()

	h.waitFor(t, "Close the rabbitmq connection.", "broker.example:5672")
	for _, e := range h.snapshot() {
		if strings.Contains(e, "SENTINEL") {
			t.Errorf("log entry contains the credential: %s", e)
		}
	}
}

// Test_connect_no_credentials runs connect() against addresses that can not be reached. The retry
// loop of connect() can not be stopped, so its goroutines end with the test process; every
// assertion is made on the entries collected by the shared hook.
func Test_connect_no_credentials(t *testing.T) {
	h := installCaptureHook(t)

	// well formed but refused: the host is logged, the credential is not
	r1 := &rabbit{uri: "amqp://user:SENTINEL@127.0.0.1:1/"}
	go r1.connect()
	h.waitFor(t, "Could not connect to rabbitmq", "127.0.0.1:1")

	// malformed: the dial error embeds the whole address, so the fixed text is logged instead
	for _, uri := range []string{
		"amqp://user:SENTINEL@broker.example:99x/",
		"amqp://user:SENTINEL@broker.example/%zz",
		"amqp://user:SENTINEL%zz@broker.example/",
	} {
		r := &rabbit{uri: uri}
		go r.connect()
	}
	h.waitFor(t, "Could not connect to rabbitmq", errInvalidURI.Error())

	// give the other goroutines time to log their first attempt
	time.Sleep(300 * time.Millisecond)
	for _, e := range h.snapshot() {
		if strings.Contains(e, "SENTINEL") {
			t.Errorf("log entry contains the credential: %s", e)
		}
	}
}
