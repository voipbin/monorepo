package main

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// isolateLogger swaps the global logger hooks for a test hook and restores the
// previous hooks when the test ends. logrus has no RemoveHook.
func isolateLogger(t *testing.T) *logrustest.Hook {
	t.Helper()

	oldHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(oldHooks) })

	return logrustest.NewGlobal()
}

// waitLogEntry polls the hook until an entry whose message contains all of the
// fragments appears, or fails the test after the upper bound.
func waitLogEntry(t *testing.T, hook *logrustest.Hook, fragments ...string) *logrus.Entry {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, e := range hook.AllEntries() {
			matched := true
			for _, f := range fragments {
				if !strings.Contains(e.Message, f) {
					matched = false
					break
				}
			}
			if matched {
				return e
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("Timed out waiting for a log entry containing %q", fragments)
	return nil
}

var configEnvKeys = []string{
	"RABBITMQ_ADDRESS",
	"RABBITMQ_QUEUE_LISTEN",
	"INTERFACE_NAME",
	"PROMETHEUS_ENDPOINT",
	"PROMETHEUS_LISTEN_ADDRESS",
	"SIP_TIMEOUT",
}

type configSnapshot struct {
	rabbitMQAddress         string
	rabbitMQQueueListen     string
	interfaceName           string
	prometheusEndpoint      string
	prometheusListenAddress string
	sipTimeout              time.Duration
}

func snapshotConfig() configSnapshot {
	return configSnapshot{
		rabbitMQAddress:         rabbitMQAddress,
		rabbitMQQueueListen:     rabbitMQQueueListen,
		interfaceName:           interfaceName,
		prometheusEndpoint:      prometheusEndpoint,
		prometheusListenAddress: prometheusListenAddress,
		sipTimeout:              sipTimeout,
	}
}

// runInitVariable calls initVariable against a fresh flag set with the given
// command line and environment, and returns the resulting configuration. The
// package globals, pflag.CommandLine and os.Args are restored afterwards.
// viper has no way to restore its global state; viper.Reset is used so earlier
// bindings do not leak in, and nothing else in this package reads viper.
func runInitVariable(t *testing.T, args []string, env map[string]string) configSnapshot {
	t.Helper()

	saved := snapshotConfig()
	savedFlags := pflag.CommandLine
	savedArgs := os.Args
	t.Cleanup(func() {
		rabbitMQAddress = saved.rabbitMQAddress
		rabbitMQQueueListen = saved.rabbitMQQueueListen
		interfaceName = saved.interfaceName
		prometheusEndpoint = saved.prometheusEndpoint
		prometheusListenAddress = saved.prometheusListenAddress
		sipTimeout = saved.sipTimeout
		pflag.CommandLine = savedFlags
		os.Args = savedArgs
	})

	// Blank every key first so the host environment cannot leak in. viper
	// treats an empty variable as unset.
	for _, k := range configEnvKeys {
		t.Setenv(k, "")
	}
	for k, v := range env {
		t.Setenv(k, v)
	}

	pflag.CommandLine = pflag.NewFlagSet("test", pflag.ContinueOnError)
	os.Args = append([]string{"x"}, args...)
	viper.Reset()

	initVariable()

	return snapshotConfig()
}

func Test_initVariable_defaults(t *testing.T) {
	got := runInitVariable(t, nil, nil)

	want := configSnapshot{
		rabbitMQAddress:         defaultRabbitMQAddress,
		rabbitMQQueueListen:     defaultRabbitMQQueueListen,
		interfaceName:           defaultInterfaceName,
		prometheusEndpoint:      defaultPrometheusEndpoint,
		prometheusListenAddress: defaultPrometheusListenAddress,
		sipTimeout:              5 * time.Second,
	}
	if got != want {
		t.Errorf("Wrong defaults.\nexpect: %+v\ngot: %+v", want, got)
	}
}

func Test_initVariable_env(t *testing.T) {
	got := runInitVariable(t, nil, map[string]string{
		"RABBITMQ_ADDRESS":          "amqp://env-rabbit.invalid:5672",
		"RABBITMQ_QUEUE_LISTEN":     "env.queue.request",
		"INTERFACE_NAME":            "envif0",
		"PROMETHEUS_ENDPOINT":       "/env-metrics",
		"PROMETHEUS_LISTEN_ADDRESS": ":3111",
		"SIP_TIMEOUT":               "3s",
	})

	want := configSnapshot{
		rabbitMQAddress:         "amqp://env-rabbit.invalid:5672",
		rabbitMQQueueListen:     "env.queue.request",
		interfaceName:           "envif0",
		prometheusEndpoint:      "/env-metrics",
		prometheusListenAddress: ":3111",
		sipTimeout:              3 * time.Second,
	}
	if got != want {
		t.Errorf("Wrong env configuration.\nexpect: %+v\ngot: %+v", want, got)
	}
}

func Test_initVariable_flags(t *testing.T) {
	got := runInitVariable(t, []string{
		"--rabbitmq_address=amqp://flag-rabbit.invalid:5672",
		"--rabbitmq_queue_listen=flag.queue.request",
		"--interface_name=flagif0",
		"--prometheus_endpoint=/flag-metrics",
		"--prometheus_listen_address=:4222",
		"--sip_timeout=7s",
	}, nil)

	want := configSnapshot{
		rabbitMQAddress:         "amqp://flag-rabbit.invalid:5672",
		rabbitMQQueueListen:     "flag.queue.request",
		interfaceName:           "flagif0",
		prometheusEndpoint:      "/flag-metrics",
		prometheusListenAddress: ":4222",
		sipTimeout:              7 * time.Second,
	}
	if got != want {
		t.Errorf("Wrong flag configuration.\nexpect: %+v\ngot: %+v", want, got)
	}
}

func Test_initVariable_flagOverEnv(t *testing.T) {
	got := runInitVariable(t,
		[]string{"--interface_name=flagif0", "--sip_timeout=7s"},
		map[string]string{"INTERFACE_NAME": "envif0", "SIP_TIMEOUT": "3s"},
	)

	if got.interfaceName != "flagif0" {
		t.Errorf("Expected the flag to win over the environment, got interface %q", got.interfaceName)
	}
	if got.sipTimeout != 7*time.Second {
		t.Errorf("Expected the flag to win over the environment, got timeout %v", got.sipTimeout)
	}
}

func Test_initVariable_invalidSIPTimeout(t *testing.T) {
	tests := []struct {
		name string
		env  string
	}{
		{"not a duration", "bogus"},
		{"missing unit", "30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := isolateLogger(t)

			got := runInitVariable(t, nil, map[string]string{"SIP_TIMEOUT": tt.env})
			if got.sipTimeout != 5*time.Second {
				t.Errorf("Expected the 5s fallback, got %v", got.sipTimeout)
			}

			e := waitLogEntry(t, hook, "Could not parse sip_timeout", tt.env)
			if e.Level != logrus.WarnLevel {
				t.Errorf("Expected warn level, got %v", e.Level)
			}
		})
	}
}

var promEndpointSeq atomic.Int64

// Test_initProm_listenError verifies that a listener failure is reported on
// the error log. The address is one that can never be bound, so the result does
// not depend on which ports are free on the host.
func Test_initProm_listenError(t *testing.T) {
	hook := isolateLogger(t)

	// The default mux rejects a repeated pattern, so use a new path per call.
	endpoint := fmt.Sprintf("/metrics-test-%d", promEndpointSeq.Add(1))
	initProm(endpoint, "127.0.0.1:99999")

	e := waitLogEntry(t, hook, "Could not start prometheus server", "99999")
	if e.Level != logrus.ErrorLevel {
		t.Errorf("Expected error level, got %v", e.Level)
	}
}
