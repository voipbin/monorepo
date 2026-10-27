package main

import (
	"errors"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-queue-manager/internal/config"
	"monorepo/bin-queue-manager/pkg/queuecallhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

// promEndpointSeq makes every endpoint unique within the process, because the
// endpoints are registered on the shared default mux, which rejects a pattern
// registered twice, and -count repeats the tests inside one process.
var promEndpointSeq atomic.Int64

func nextPromEndpoint() string {
	return "/metrics-test-" + strconv.FormatInt(promEndpointSeq.Add(1), 10)
}

// logCollector is a logrus hook that forwards every entry to a channel.
type logCollector struct {
	ch chan *logrus.Entry
}

func (c *logCollector) Levels() []logrus.Level { return logrus.AllLevels }

func (c *logCollector) Fire(e *logrus.Entry) error {
	select {
	case c.ch <- e:
	default:
	}
	return nil
}

// collectLogs installs the collector on the standard logger and restores the
// previous hooks afterwards.
func collectLogs(t *testing.T) *logCollector {
	t.Helper()

	c := &logCollector{ch: make(chan *logrus.Entry, 1024)}
	prev := make(logrus.LevelHooks)
	for level, hooks := range logrus.StandardLogger().Hooks {
		prev[level] = append([]logrus.Hook{}, hooks...)
	}

	logrus.AddHook(c)
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(prev) })

	return c
}

// waitMessage waits until an entry whose message contains the given text is logged.
func (c *logCollector) waitMessage(t *testing.T, text string) {
	t.Helper()

	timeout := time.After(5 * time.Second)
	for {
		select {
		case e := <-c.ch:
			if strings.Contains(e.Message, text) {
				return
			}
		case <-timeout:
			t.Fatalf("the log message %q was not written", text)
		}
	}
}

// count drains the collected entries and returns how many contain the text.
func (c *logCollector) count(text string) int {
	n := 0
	for {
		select {
		case e := <-c.ch:
			if strings.Contains(e.Message, text) {
				n++
			}
		default:
			return n
		}
	}
}

// useUnparsableInfra points the global config at addresses that are rejected while they
// are parsed, so that nothing is ever dialed, and restores the previous config afterwards.
func useUnparsableInfra(t *testing.T) {
	t.Helper()

	prevFormatter := logrus.StandardLogger().Formatter
	prevLevel := logrus.GetLevel()
	prev := *config.Get()

	*config.Get() = config.Config{
		DatabaseDSN:             "not-a-dsn",
		PrometheusEndpoint:      nextPromEndpoint(),
		PrometheusListenAddress: "no-port",
		RabbitMQAddress:         "no-port",
		RedisAddress:            "no-port",
	}

	t.Cleanup(func() {
		*config.Get() = prev
		logrus.SetFormatter(prevFormatter)
		logrus.SetLevel(prevLevel)
		viper.Reset()
	})
}

func Test_initProm_listenFails(t *testing.T) {
	logs := collectLogs(t)

	initProm(nextPromEndpoint(), "no-port")

	logs.waitMessage(t, "Prometheus server error")
}

func Test_runDaemon_databaseFails(t *testing.T) {
	useUnparsableInfra(t)
	logs := collectLogs(t)

	// runDaemon starts the signal notifier. Feed it one value afterwards so that the
	// goroutine it created is released, and stop the notification of the process.
	t.Cleanup(func() {
		signal.Stop(chSigs)
		chSigs <- syscall.SIGTERM
		select {
		case <-chDone:
		case <-time.After(5 * time.Second):
			t.Errorf("the signal notifier did not release")
		}
	})

	err := runDaemon()
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	// the database address is rejected while it is parsed, so no connection is tried.
	for _, s := range []string{"could not connect to database", "invalid DSN"} {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("Wrong match. expect the error to contain %q, got: %v", s, err)
		}
	}

	logs.waitMessage(t, "Prometheus server error")
}

func Test_runListen_runFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	logs := collectLogs(t)

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockQueue := queuehandler.NewMockQueueHandler(mc)
	mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

	// nothing else is expected after the failed queue creation.
	mockSock.EXPECT().QueueCreate("bin-manager.queue-manager.request", "normal").Return(errors.New("simulated queue failure"))

	// runListen only logs the failure of the listener and keeps running.
	if err := runListen(mockSock, mockQueue, mockQueuecall); err != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", err)
	}

	if got := logs.count("Could not run the listenhandler correctly"); got != 1 {
		t.Errorf("Wrong match. expect: 1 failure log, got: %d", got)
	}
}

func Test_main_help(t *testing.T) {
	prevArgs := os.Args
	prevFormatter := logrus.StandardLogger().Formatter
	prevLevel := logrus.GetLevel()
	prevConfig := *config.Get()

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("could not open the null device. err: %v", err)
	}
	prevStdout := os.Stdout

	t.Cleanup(func() {
		os.Args = prevArgs
		os.Stdout = prevStdout
		_ = devNull.Close()
		*config.Get() = prevConfig
		logrus.SetFormatter(prevFormatter)
		logrus.SetLevel(prevLevel)
		viper.Reset()
	})

	os.Args = []string{"queue-manager", "--help"}
	os.Stdout = devNull

	// --help is answered by cobra before the daemon starts, so main returns
	// instead of exiting.
	main()
}
