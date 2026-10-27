package main

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"monorepo/bin-message-manager/internal/config"
)

// promPathSeq makes every registered metrics path unique. The default mux cannot unregister a pattern
// and a duplicated registration panics, so a repeated run (-count, -shuffle) needs a new path each time.
var promPathSeq atomic.Int64

// uniquePromPath returns a metrics path that was never registered in this process.
func uniquePromPath() string {
	return "/metrics-test-" + strconv.FormatInt(promPathSeq.Add(1), 10)
}

// logSignalHook forwards the messages accepted by the filter to a channel, without blocking the logger.
type logSignalHook struct {
	filter func(msg string) bool
	ch     chan string
}

func (h *logSignalHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *logSignalHook) Fire(e *logrus.Entry) error {
	if !h.filter(e.Message) {
		return nil
	}

	select {
	case h.ch <- e.Message:
	default:
	}
	return nil
}

// installLogSignalHook replaces the standard logger hooks and restores them, and the log level, after the test.
func installLogSignalHook(t *testing.T, filter func(msg string) bool) *logSignalHook {
	t.Helper()

	hook := &logSignalHook{filter: filter, ch: make(chan string, 1)}
	origLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	origHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	logrus.AddHook(hook)

	t.Cleanup(func() {
		logrus.StandardLogger().ReplaceHooks(origHooks)
		logrus.SetLevel(origLevel)
	})
	return hook
}

// waitLog receives the forwarded message or fails the test after signalWait.
func (h *logSignalHook) waitLog(t *testing.T) string {
	t.Helper()

	select {
	case msg := <-h.ch:
		return msg
	case <-time.After(signalWait):
		t.Fatalf("Wrong match. expect: log message, got: timeout")
		return ""
	}
}

func Test_initProm_enabled(t *testing.T) {
	t.Run("serves the metrics on the listen address", func(t *testing.T) {
		path := uniquePromPath()
		hook := installLogSignalHook(t, func(msg string) bool {
			return strings.Contains(msg, "starting on") && strings.HasSuffix(msg, path)
		})

		initProm(path, "127.0.0.1:0")
		hook.waitLog(t)

		req := httptest.NewRequest(http.MethodGet, path, nil)
		h, pattern := http.DefaultServeMux.Handler(req)
		if pattern != path {
			t.Fatalf("Wrong match. expect: %s, got: %s", path, pattern)
		}

		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Wrong status code. expect: %d, got: %d", http.StatusOK, w.Code)
		}
		// the registered handler is the prometheus exposition handler, not an arbitrary one.
		if !strings.Contains(w.Body.String(), "# HELP") {
			t.Errorf("Wrong body. expect: prometheus metrics output, got: %s", w.Body.String())
		}
	})

	t.Run("listen address is already in use", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Could not listen. err: %v", err)
		}
		t.Cleanup(func() { _ = l.Close() })
		addr := l.Addr().String()

		hook := installLogSignalHook(t, func(msg string) bool {
			return strings.Contains(msg, "Prometheus server error") && strings.Contains(msg, addr)
		})

		initProm(uniquePromPath(), addr)

		msg := hook.waitLog(t)
		if !strings.Contains(msg, "address already in use") {
			t.Errorf("Wrong match. expect: address already in use, got: %s", msg)
		}
	})
}

// captureMainStdout runs fn and returns what it wrote to os.Stdout. The output must stay smaller than the
// pipe buffer because it is read after fn returns.
func captureMainStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Could not create the pipe. err: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = orig
		_ = r.Close()
	})

	fn()
	_ = w.Close()
	os.Stdout = orig

	out, errRead := io.ReadAll(r)
	if errRead != nil {
		t.Fatalf("Could not read the pipe. err: %v", errRead)
	}
	return string(out)
}

// Test_main_help runs main with the help flag only. The help flag is handled before the persistent
// pre-run hook, so the config loader does not run. Any other argument can end the test binary through
// os.Exit, which hides the real failure.
func Test_main_help(t *testing.T) {
	origArgs := os.Args
	origCfg := *config.Get()
	origLevel := logrus.GetLevel()
	origFormatter := logrus.StandardLogger().Formatter
	viper.Reset()
	t.Cleanup(func() {
		os.Args = origArgs
		viper.Reset()
		*config.Get() = origCfg
		logrus.SetLevel(origLevel)
		logrus.SetFormatter(origFormatter)
	})
	os.Args = []string{"message-manager", "--help"}

	out := captureMainStdout(t, main)

	if expect := "message-manager [flags]"; !strings.Contains(out, expect) {
		t.Errorf("Wrong match. expect: help output contains %q, got: %s", expect, out)
	}
	if expect := "--database_dsn"; !strings.Contains(out, expect) {
		t.Errorf("Wrong match. expect: help output contains %q, got: %s", expect, out)
	}
}
