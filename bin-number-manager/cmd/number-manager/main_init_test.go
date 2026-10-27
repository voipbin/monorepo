package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"monorepo/bin-number-manager/internal/config"
)

// closedPortDSN points at a loopback port without a listener, so a connection attempt is refused at once.
const closedPortDSN = "svc@tcp(127.0.0.1:1)/testdb?timeout=1s"

// closedPortAddr is a loopback address without a listener.
const closedPortAddr = "127.0.0.1:1"

// promSeq makes every metrics endpoint of the tests unique: http.DefaultServeMux panics on a
// pattern that is registered twice, which -count and -shuffle would otherwise trigger.
var promSeq atomic.Int64

func uniqueEndpoint() string {
	return fmt.Sprintf("/test-metrics-%d", promSeq.Add(1))
}

// setTestConfig replaces the process wide config for one test and restores it afterwards.
func setTestConfig(t *testing.T, c config.Config) {
	t.Helper()

	config.LoadGlobalConfig()
	orig := *config.Get()
	*config.Get() = c
	t.Cleanup(func() { *config.Get() = orig })
}

// chanHook forwards the log messages of the standard logger to a channel.
type chanHook struct {
	ch chan string
}

func (h *chanHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *chanHook) Fire(e *logrus.Entry) error {
	select {
	case h.ch <- e.Message:
	default:
	}
	return nil
}

// captureLog installs a hook on the standard logger for one test. The previous hooks and level are restored.
func captureLog(t *testing.T) *chanHook {
	t.Helper()

	std := logrus.StandardLogger()
	hook := &chanHook{ch: make(chan string, 64)}
	origLevel := std.GetLevel()
	origHooks := std.ReplaceHooks(make(logrus.LevelHooks))
	std.AddHook(hook)
	std.SetLevel(logrus.InfoLevel)
	t.Cleanup(func() {
		std.ReplaceHooks(origHooks)
		std.SetLevel(origLevel)
	})

	return hook
}

// waitLog waits for a logged message that contains all of the given parts.
func (h *chanHook) waitLog(t *testing.T, parts ...string) string {
	t.Helper()

	deadline := time.After(waitTimeout)
	for {
		select {
		case msg := <-h.ch:
			ok := true
			for _, p := range parts {
				if !strings.Contains(msg, p) {
					ok = false
					break
				}
			}
			if ok {
				return msg
			}
		case <-deadline:
			t.Fatalf("timed out waiting for a log message containing: %v", parts)
		}
	}
}

// endSignalGoroutine ends, at the end of the test, the goroutine that initSignal starts: one
// SIGTERM is pushed through the channel (no real signal) and the done channel is drained.
func endSignalGoroutine(t *testing.T) {
	t.Helper()

	for len(chDone) > 0 {
		<-chDone
	}
	t.Cleanup(func() {
		signal.Stop(chSigs)
		chSigs <- syscall.SIGTERM
		select {
		case <-chDone:
		case <-time.After(waitTimeout):
			t.Errorf("the signal goroutine did not end")
		}
	})
}

func Test_initCache(t *testing.T) {
	t.Run("connected", func(t *testing.T) {
		mr := miniredis.RunT(t)
		setTestConfig(t, config.Config{RedisAddress: mr.Addr()})

		res, err := initCache()
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res == nil {
			t.Errorf("Wrong match. expect: a cache handler, got: nil")
		}
	})

	t.Run("unreachable", func(t *testing.T) {
		setTestConfig(t, config.Config{RedisAddress: closedPortAddr})

		res, err := initCache()
		if err == nil || !strings.Contains(err.Error(), "cache connect error") {
			t.Errorf("Wrong match. expect error containing: cache connect error, got: %v", err)
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
	})
}

func Test_initProm(t *testing.T) {
	t.Run("registers the endpoint and starts the server", func(t *testing.T) {
		hook := captureLog(t)
		endpoint := uniqueEndpoint()

		initProm(endpoint, "127.0.0.1:0")

		req := httptest.NewRequest(http.MethodGet, endpoint, nil)
		if _, pattern := http.DefaultServeMux.Handler(req); pattern != endpoint {
			t.Errorf("Wrong match. expect the pattern: %s, got: %q", endpoint, pattern)
		}
		hook.waitLog(t, "Prometheus metrics server starting on 127.0.0.1:0"+endpoint)
	})

	t.Run("logs a listen error", func(t *testing.T) {
		hook := captureLog(t)

		initProm(uniqueEndpoint(), "127.0.0.1:-1")

		hook.waitLog(t, "Prometheus server error")
	})
}

func Test_runDaemon_databaseUnreachable(t *testing.T) {
	hook := captureLog(t)
	endSignalGoroutine(t)

	endpoint := uniqueEndpoint()
	setTestConfig(t, config.Config{
		PrometheusEndpoint:      endpoint,
		PrometheusListenAddress: "127.0.0.1:0",
		DatabaseDSN:             closedPortDSN,
	})

	err := runDaemon()
	if err == nil || !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong match. expect error containing: could not connect to the database, got: %v", err)
	}

	// the startup log is written before the database is opened
	hook.waitLog(t, "Starting number-manager")
}

func Test_main_help(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	origLevel := logrus.GetLevel()
	origFormatter := logrus.StandardLogger().Formatter
	t.Cleanup(func() {
		logrus.SetLevel(origLevel)
		logrus.SetFormatter(origFormatter)
	})

	origArgs := os.Args
	os.Args = []string{"number-manager", "--help"}
	t.Cleanup(func() { os.Args = origArgs })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe. err: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = origStdout })

	main()

	os.Stdout = origStdout
	if errClose := w.Close(); errClose != nil {
		t.Fatalf("could not close the pipe. err: %v", errClose)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("could not read the pipe. err: %v", err)
	}

	if !strings.Contains(string(out), "Voipbin Number Manager Daemon") {
		t.Errorf("Wrong match. expect the help to contain: Voipbin Number Manager Daemon, got: %s", out)
	}
}
