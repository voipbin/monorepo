package main

import (
	"net/http"
	"os"
	"os/signal"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

// exitCalled is the panic value that replaces the process exit while a test runs the daemon.
type exitCalled struct{ code int }

// useFreshMux replaces the default HTTP mux, because the metrics path can be registered only
// once on a mux and the tests may be repeated in one process.
func useFreshMux(t *testing.T) {
	t.Helper()

	orig := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	t.Cleanup(func() { http.DefaultServeMux = orig })
}

// useLogHook records the log entries of the test and restores the logger when the test ends.
func useLogHook(t *testing.T) *logrustest.Hook {
	t.Helper()

	orig := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	hook := logrustest.NewGlobal()
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(orig) })

	return hook
}

// findLog waits until the hook has seen an entry whose message contains want. The wait is only
// an upper bound for the goroutine that writes the entry; it does not decide the result.
func findLog(t *testing.T, hook *logrustest.Hook, level logrus.Level, want string) *logrus.Entry {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, e := range hook.AllEntries() {
			if e.Level == level && strings.Contains(e.Message, want) {
				return e
			}
		}
		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf("no %v log entry containing %q within the timeout", level, want)

	return nil
}

// Test_initProm_listenError checks that a listener that cannot start is logged and does not stop
// the process. The address has no port, so the listener fails at once and offline.
func Test_initProm_listenError(t *testing.T) {
	useFreshMux(t)
	hook := useLogHook(t)

	initProm("/test-initprom-metrics", "initprom-address-without-port")

	findLog(t, hook, logrus.InfoLevel, "Prometheus metrics server starting on initprom-address-without-port/test-initprom-metrics")

	// the address is part of the message, so a log of an earlier test cannot satisfy this wait
	failed := findLog(t, hook, logrus.ErrorLevel, "initprom-address-without-port")
	if !strings.Contains(failed.Message, "missing port in address") {
		t.Errorf("failure message = %q, want the listen error", failed.Message)
	}
}

// Test_main_databaseUnavailable runs the whole command with a database address that the driver
// rejects while parsing it. The daemon must stop at the database connection with a fatal log,
// before it opens RabbitMQ or Redis.
func Test_main_databaseUnavailable(t *testing.T) {
	useFreshMux(t)
	hook := useLogHook(t)

	// the config is loaded once per process, so every test of main() must use these values
	t.Setenv("DATABASE_DSN", "bad")
	t.Setenv("PROMETHEUS_ENDPOINT", "/test-main-metrics")
	t.Setenv("PROMETHEUS_LISTEN_ADDRESS", "main-address-without-port")

	origArgs := os.Args
	os.Args = []string{"talk-manager"}
	t.Cleanup(func() { os.Args = origArgs })

	origExit := logrus.StandardLogger().ExitFunc
	logrus.StandardLogger().ExitFunc = func(code int) { panic(exitCalled{code}) }
	t.Cleanup(func() { logrus.StandardLogger().ExitFunc = origExit })

	// runDaemon starts the signal handling; stop it and empty the channels when the test ends
	t.Cleanup(func() {
		signal.Stop(chSigs)
		drain()
	})

	var got any
	func() {
		defer func() { got = recover() }()
		main()
	}()

	exit, ok := got.(exitCalled)
	if !ok {
		t.Fatalf("recovered %v, want the exit of the fatal log", got)
	}
	if exit.code != 1 {
		t.Errorf("exit code = %d, want 1", exit.code)
	}

	var fatal *logrus.Entry
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.FatalLevel {
			fatal = e
		}
		if strings.Contains(e.Message, "RabbitMQ initialized") || strings.Contains(e.Message, "Redis initialized") {
			t.Errorf("the daemon went past the database connection: %q", e.Message)
		}
	}
	if fatal == nil || !strings.Contains(fatal.Message, "Could not connect to the database") {
		t.Errorf("fatal log entry = %+v, want the database connection failure", fatal)
	}
}
