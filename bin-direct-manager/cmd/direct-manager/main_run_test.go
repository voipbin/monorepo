package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-direct-manager/internal/config"
)

// invalidDSN is rejected by the database driver while the connection is opened, so the
// tests below fail at the connect step without any network access.
const invalidDSN = "not-a-dsn"

// metricsSeq makes every metrics endpoint unique, because initProm registers its pattern on
// the process wide http.DefaultServeMux and a repeated pattern panics.
var metricsSeq atomic.Int64

func uniqueEndpoint() string {
	return fmt.Sprintf("/metrics-test-%d", metricsSeq.Add(1))
}

// resetState isolates a test from the process wide viper and config state and from the
// environment. Tests in this package must not run in parallel.
func resetState(t *testing.T) {
	t.Helper()

	prev := *config.Get()
	viper.Reset()
	for _, name := range []string{
		"DATABASE_DSN", "PROMETHEUS_ENDPOINT", "PROMETHEUS_LISTEN_ADDRESS",
		"RABBITMQ_ADDRESS", "REDIS_ADDRESS", "REDIS_DATABASE", "REDIS_PASSWORD",
	} {
		t.Setenv(name, "")
	}

	t.Cleanup(func() {
		viper.Reset()
		*config.Get() = prev
	})
}

// newServiceCommand returns a command that carries the service flags with the given values
// marked as changed, so that a flag value always wins over the environment.
func newServiceCommand(t *testing.T, dsn string, endpoint string, listen string) *cobra.Command {
	t.Helper()

	cmd := &cobra.Command{Use: "test"}
	flags := cmd.Flags()
	flags.String("database_dsn", "", "")
	flags.String("prometheus_endpoint", "", "")
	flags.String("prometheus_listen_address", "", "")
	flags.String("rabbitmq_address", "", "")
	flags.String("redis_address", "", "")
	flags.Int("redis_database", 1, "")
	flags.String("redis_password", "", "")

	for name, value := range map[string]string{
		"database_dsn":              dsn,
		"prometheus_endpoint":       endpoint,
		"prometheus_listen_address": listen,
	} {
		if err := flags.Set(name, value); err != nil {
			t.Fatalf("could not set the flag %s: %v", name, err)
		}
	}

	return cmd
}

// entryHook forwards the log entries to a channel without ever blocking the logger.
type entryHook struct {
	entries chan *logrus.Entry
}

func (h *entryHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *entryHook) Fire(e *logrus.Entry) error {
	select {
	case h.entries <- e:
	default:
	}
	return nil
}

func newEntryHook(t *testing.T) *entryHook {
	t.Helper()

	h := &entryHook{entries: make(chan *logrus.Entry, 256)}
	logrus.AddHook(h)
	t.Cleanup(func() {
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	})

	return h
}

// waitEntry waits, with an upper bound, for a log entry of the given level whose message
// contains the text.
func (h *entryHook) waitEntry(t *testing.T, level logrus.Level, text string) *logrus.Entry {
	t.Helper()

	timeout := time.After(5 * time.Second)
	for {
		select {
		case e := <-h.entries:
			if e.Level == level && strings.Contains(e.Message, text) {
				return e
			}
		case <-timeout:
			t.Fatalf("no %v log entry containing %q within the time limit", level, text)
			return nil
		}
	}
}

func Test_run_configError(t *testing.T) {
	resetState(t)

	// a command without the service flags cannot be bound; run returns before it touches
	// anything else, including the metrics listener
	err := run(&cobra.Command{Use: "test"}, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to initialize config") {
		t.Errorf("run() error = %v, want the config initialization context", err)
	}
}

func Test_run_databaseError(t *testing.T) {
	resetState(t)

	cmd := newServiceCommand(t, invalidDSN, uniqueEndpoint(), "127.0.0.1:0")

	err := run(cmd, nil)
	if err == nil {
		t.Fatalf("run() error = nil, want the database connect failure")
	}
	// the connect stage text, which also tells it apart from the migration stage failing on
	// the same dsn
	if !strings.Contains(err.Error(), "database open error") {
		t.Errorf("run() error = %q, want the database connect stage text", err.Error())
	}
	if strings.Contains(err.Error(), "could not parse the database dsn") {
		t.Errorf("run() error = %q, want it to stop before the migration stage", err.Error())
	}
}

func Test_initProm_retriesWhileBusy(t *testing.T) {
	hook := newEntryHook(t)

	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not occupy a loopback port: %v", err)
	}
	addr := busy.Addr().String()
	endpoint := uniqueEndpoint()

	// the goroutine started here cannot be stopped; it ends with the process
	initProm(endpoint, addr)

	hook.waitEntry(t, logrus.ErrorLevel, "Could not start prometheus listener")

	if errClose := busy.Close(); errClose != nil {
		t.Fatalf("could not release the port: %v", errClose)
	}

	// after the port is free the retry loop must come up and serve the metrics endpoint
	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(10 * time.Second)
	lastStatus := 0
	for time.Now().Before(deadline) {
		res, errGet := client.Get("http://" + addr + endpoint)
		if errGet == nil {
			lastStatus = res.StatusCode
			_ = res.Body.Close()
			if lastStatus == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("the metrics endpoint never served after the port was released. last status: %d", lastStatus)
}
