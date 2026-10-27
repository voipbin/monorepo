package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/viper"

	"monorepo/bin-schedule-manager/internal/config"
)

// metricsEndpointSeq makes every initProm call register a distinct pattern:
// the default mux panics when the same pattern is registered twice.
var metricsEndpointSeq atomic.Int32

func nextMetricsEndpoint() string {
	return fmt.Sprintf("/test-metrics-%d", metricsEndpointSeq.Add(1))
}

// unbindableListenAddress is a host without a port, so ListenAndServe always
// fails and nothing ever listens.
const unbindableListenAddress = "127.0.0.1"

// captureStdout runs fn and returns what it wrote to os.Stdout. The pipe is
// drained in a goroutine so a large output cannot block the writer.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	old := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = old })

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	_ = w.Close()
	os.Stdout = old

	return <-done
}

func Test_initProm_server_error(t *testing.T) {
	oldHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(oldHooks) })
	hook := logtest.NewGlobal()

	initProm(nextMetricsEndpoint(), unbindableListenAddress)

	deadline := time.Now().Add(testWait)
	for time.Now().Before(deadline) {
		for _, e := range hook.AllEntries() {
			if e.Level == logrus.ErrorLevel && strings.Contains(e.Message, "Prometheus server error") {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("Wrong match. expect: the server error is logged, got: timeout")
}

// Test_runDaemon_database_connect_error runs the daemon with a DSN that does
// not parse: it must stop at the database step with the connect error, and
// the parse error must stay reachable through the wrapping.
func Test_runDaemon_database_connect_error(t *testing.T) {
	const badDSN = "not-a-dsn" // no slash, so the driver returns its package-level sentinel

	_, expectErr := gomysql.ParseDSN(badDSN)
	if expectErr == nil {
		t.Fatalf("Wrong match. expect: the DSN is rejected by the driver, got: nil")
	}

	// runDaemon starts the signal goroutine; release it and drain chDone so the
	// package channels are empty for the other tests
	t.Cleanup(func() {
		signal.Stop(chSigs)
		chSigs <- syscall.SIGTERM
		select {
		case <-chDone:
		case <-time.After(testWait):
			t.Errorf("Wrong match. expect: the signal goroutine finishes, got: timeout")
		}
	})

	config.LoadGlobalConfig()
	old := *config.Get()
	*config.Get() = config.Config{
		PrometheusEndpoint:      nextMetricsEndpoint(),
		PrometheusListenAddress: unbindableListenAddress,
		DatabaseDSN:             badDSN,
	}
	t.Cleanup(func() { *config.Get() = old })

	err := runDaemon()
	if err == nil {
		t.Fatalf("Wrong match. expect: an error, got: nil")
	}
	if !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong match. expect: the connect error message, got: %v", err)
	}
	if !errors.Is(err, expectErr) {
		t.Errorf("Wrong match. expect: the driver parse error reachable, got: %v", err)
	}
}

func Test_main_help(t *testing.T) {
	oldArgs := os.Args
	oldFormatter := logrus.StandardLogger().Formatter
	oldLevel := logrus.GetLevel()
	t.Cleanup(func() {
		os.Args = oldArgs
		logrus.SetFormatter(oldFormatter)
		logrus.SetLevel(oldLevel)
		viper.Reset()
	})

	os.Args = []string{"schedule-manager", "--help"}

	out := captureStdout(t, main)

	for _, expect := range []string{"Voipbin Schedule Manager Daemon", "--database_dsn", "--schedule_backup_dir"} {
		if !strings.Contains(out, expect) {
			t.Errorf("Wrong match. expect: the usage contains %q, got: %q", expect, out)
		}
	}
}
