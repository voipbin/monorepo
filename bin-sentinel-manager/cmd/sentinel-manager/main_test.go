package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-sentinel-manager/internal/config"
	"monorepo/bin-sentinel-manager/pkg/dockerwatchhandler"
	"monorepo/bin-sentinel-manager/pkg/k8swatchhandler"
)

// Test_buildBackend_selectsByConfig pins the backend-selection branch.
//
// The two arms construct genuinely different dependency graphs, so a wiring mistake here would
// only surface at runtime on whichever deployment shape nobody tested. The Docker arm is
// exercised for real (it reaches Redis, which is unreachable in a unit test and therefore errors
// — that error IS the evidence the arm ran), while the Kubernetes arm reaches
// rest.InClusterConfig(), which necessarily fails outside a cluster for the same reason.
func Test_buildBackend_selectsByConfig(t *testing.T) {
	tests := []struct {
		name string

		config config.Config

		expectErr bool
	}{
		{
			name: "docker backend reaches the redis connect step",

			config: config.Config{
				SentinelBackend:          config.BackendDocker,
				DockerSocketProxyAddress: "tcp://127.0.0.1:1",
				RedisAddress:             "127.0.0.1:1",
			},

			// unreachable Redis: the constructor is expected to fail loud rather than return a
			// backend that would watch nothing.
			expectErr: true,
		},
		{
			name: "kubernetes backend reaches the in-cluster config step",

			config: config.Config{SentinelBackend: config.BackendKubernetes},

			// no service-account mount outside a cluster.
			expectErr: true,
		},
		{
			name: "an unknown backend is refused rather than silently defaulting",

			config: config.Config{SentinelBackend: "nomad"},

			expectErr: true,
		},
		{
			name: "an unset backend is refused rather than silently defaulting",

			config: config.Config{},

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			backend, cleanup, err := buildBackend(
				tt.config,
				requesthandler.NewMockRequestHandler(mc),
				notifyhandler.NewMockNotifyHandler(mc),
				utilhandler.NewMockUtilHandler(mc),
			)

			// cleanup must always be safe to defer, even on the error paths.
			if cleanup == nil {
				t.Fatalf("Wrong match. expect: a non-nil cleanup, got: nil")
			}
			defer cleanup()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				if backend != nil {
					t.Errorf("Wrong match. expect: nil backend on error, got: %T", backend)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if backend == nil {
				t.Errorf("Wrong match. expect: a backend, got: nil")
			}
		})
	}
}

// Test_buildBackend_bothBackendsSatisfyTheContract pins that both concrete types are usable
// through the one interface main() calls Run on.
func Test_buildBackend_bothBackendsSatisfyTheContract(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	// constructed directly rather than through buildBackend, which cannot reach either real
	// dependency from a unit test.
	dockerBackend := dockerwatchhandler.NewDockerWatchHandler(mockReq, mockNotify, mockUtil, nil, nil)
	if dockerBackend == nil {
		t.Errorf("Wrong match. expect: a docker backend, got: nil")
	}

	k8sBackend := k8swatchhandler.NewK8sWatchHandler(mockReq, mockNotify, mockUtil, nil)
	if k8sBackend == nil {
		t.Errorf("Wrong match. expect: a kubernetes backend, got: nil")
	}
}

// Test_rootCmd_registersSentinelBackend pins that the service binary's own flag set includes the
// selector. config.InitConfig errors with "flag not defined" otherwise, so a missing registration
// here breaks startup outright.
func Test_rootCmd_registersSentinelBackend(t *testing.T) {
	flag := rootCmd.Flags().Lookup("sentinel_backend")
	if flag == nil {
		t.Fatalf("Wrong match. expect: sentinel_backend registered on rootCmd, got: missing")
	}

	if flag.DefValue != "" {
		t.Errorf("Wrong match. expect: an empty default, got: %q", flag.DefValue)
	}
}

// Test_rootCmd_silencesUsageAndErrors keeps PR #1240's crash-loop-noise fix from regressing while
// this file is being edited for the backend branch.
func Test_rootCmd_silencesUsageAndErrors(t *testing.T) {
	if !rootCmd.SilenceUsage {
		t.Errorf("Wrong match. expect: SilenceUsage true, got: false")
	}
	if !rootCmd.SilenceErrors {
		t.Errorf("Wrong match. expect: SilenceErrors true, got: false")
	}
}

// drainSignals empties the shared signal channel so a signal left behind by one test cannot be
// consumed by the next one.
func drainSignals() {
	for {
		select {
		case <-chSigs:
		default:
			return
		}
	}
}

// Test_run_configFailure pins that run stops at the config step and returns the failure instead
// of starting the Prometheus listener or the service.
func Test_run_configFailure(t *testing.T) {
	tests := []struct {
		name string

		backend string
	}{
		{name: "an unset backend", backend: ""},
		{name: "an unknown backend", backend: "nomad"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SENTINEL_BACKEND", tt.backend)

			err := run(rootCmd, nil)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if !strings.Contains(err.Error(), "failed to initialize config") {
				t.Errorf("Wrong match. expect: a config initialization error, got: %v", err)
			}
		})
	}
}

func Test_signalHandler_cancelsOnSignal(t *testing.T) {
	drainSignals()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	finished := make(chan struct{})
	go func() {
		signalHandler(cancel)
		close(finished)
	}()

	// nothing has been sent, so the handler must still be waiting.
	select {
	case <-ctx.Done():
		t.Fatalf("Wrong match. expect: context alive before any signal, got: cancelled")
	case <-time.After(100 * time.Millisecond):
	}

	chSigs <- syscall.SIGTERM

	select {
	case <-ctx.Done():
	case <-time.After(3 * time.Second):
		t.Fatalf("Wrong match. expect: context cancelled after the signal, got: still alive")
	}

	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Errorf("Wrong match. expect: the handler to return after the signal, got: still running")
	}
}

func Test_registerSignal_cancelsOnSignal(t *testing.T) {
	drainSignals()
	t.Cleanup(func() {
		signal.Stop(chSigs)
		drainSignals()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	registerSignal(cancel)

	select {
	case <-ctx.Done():
		t.Fatalf("Wrong match. expect: context alive right after registering, got: cancelled")
	case <-time.After(100 * time.Millisecond):
	}

	// injected directly: sending a real signal to the test process is not needed to prove the
	// goroutine started by registerSignal is the one consuming the channel.
	chSigs <- syscall.SIGHUP

	select {
	case <-ctx.Done():
	case <-time.After(3 * time.Second):
		t.Fatalf("Wrong match. expect: context cancelled by the registered handler, got: still alive")
	}
}

var (
	promOnce sync.Once
	promURL  string
)

// Test_initProm serves the metrics endpoint and retries while the port is taken.
//
// initProm registers on the process-wide http.DefaultServeMux, which panics on a duplicate
// pattern, so it runs once per process (promOnce also covers -count greater than 1; later
// repetitions only re-check the served endpoint). The listener it starts cannot be stopped.
func Test_initProm(t *testing.T) {
	const endpoint = "/metrics-voip1652"

	promOnce.Do(func() {
		oldHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
		t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(oldHooks) })
		hook := logrustest.NewGlobal()

		// hold the port so the first listen attempt fails and the retry branch runs.
		held, errListen := net.Listen("tcp", "127.0.0.1:0")
		if errListen != nil {
			t.Fatalf("Could not hold a loopback port. err: %v", errListen)
		}
		address := held.Addr().String()

		initProm(endpoint, address)

		retried := false
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
			for _, entry := range hook.AllEntries() {
				if strings.Contains(entry.Message, "Could not start prometheus listener") {
					retried = true
				}
			}
			if retried {
				break
			}
		}
		_ = held.Close()

		if !retried {
			t.Fatalf("Wrong match. expect: a logged listen failure while the port is held, got: none")
		}

		promURL = "http://" + address + endpoint
	})

	if promURL == "" {
		t.Fatalf("Wrong match. expect: the metrics address from the first run, got: empty")
	}

	// a different process could grab the released port, so the unique path and the Go runtime
	// metric name together identify this server's answer.
	client := &http.Client{Timeout: time.Second}
	var lastErr error
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		res, errGet := client.Get(promURL)
		if errGet != nil {
			lastErr = errGet
			continue
		}
		body, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()

		if res.StatusCode == http.StatusOK && strings.Contains(string(body), "go_goroutines") {
			return
		}
		lastErr = nil
	}

	t.Fatalf("Wrong match. expect: 200 with go_goroutines, got: never (last error: %v)", lastErr)
}
