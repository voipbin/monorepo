package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-sentinel-manager/internal/config"
)

// waitTimeout bounds every wait in this file. Success paths return as soon as the awaited event
// arrives, so a generous value costs nothing and keeps slow -race runs from flaking.
const waitTimeout = 30 * time.Second

// promEndpointSeq makes every run() call register its own metrics path, because initProm
// registers on the process-wide http.DefaultServeMux and a duplicate pattern panics.
var promEndpointSeq atomic.Int64

// fakeDockerProxy is a loopback stand-in for the docker-socket-proxy. It answers the three calls
// the docker backend makes: the version ping, the container list, and the event stream.
type fakeDockerProxy struct {
	server *httptest.Server

	// eventsOpened receives one value for every event stream request that reached the server.
	eventsOpened chan struct{}
	// listRequests counts container list requests.
	listRequests atomic.Int64
}

// newFakeDockerProxy starts the server. listStatus is the status returned for the container list;
// 200 answers with an empty list, anything else with a docker-style JSON error.
func newFakeDockerProxy(t *testing.T, listStatus int) *fakeDockerProxy {
	t.Helper()

	f := &fakeDockerProxy{eventsOpened: make(chan struct{}, 16)}

	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/_ping"):
			w.Header().Set("API-Version", "1.45")
			w.WriteHeader(http.StatusOK)
			if r.Method != http.MethodHead {
				_, _ = w.Write([]byte("OK"))
			}

		case strings.HasSuffix(r.URL.Path, "/containers/json"):
			f.listRequests.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(listStatus)
			if listStatus == http.StatusOK {
				_, _ = w.Write([]byte("[]"))
				return
			}
			_, _ = w.Write([]byte(`{"message":"synthetic list failure"}`))

		case strings.HasSuffix(r.URL.Path, "/events"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			f.eventsOpened <- struct{}{}
			// hold the stream open until the client goes away.
			<-r.Context().Done()

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.server.Close)

	return f
}

// address returns the proxy address in the tcp:// form the service configuration expects.
func (f *fakeDockerProxy) address() string {
	return "tcp://" + strings.TrimPrefix(f.server.URL, "http://")
}

// waitEventsOpened blocks until the docker backend has opened its event stream.
func (f *fakeDockerProxy) waitEventsOpened(t *testing.T) {
	t.Helper()

	select {
	case <-f.eventsOpened:
	case <-time.After(waitTimeout):
		t.Fatalf("Wrong match. expect: the event stream to be opened, got: nothing within %v", waitTimeout)
	}
}

// useMockSock replaces the RabbitMQ factory with a mock for the test. Connect and the global
// topic declaration are the only sock calls the service start path makes, each exactly once.
func useMockSock(t *testing.T) {
	t.Helper()

	mc := gomock.NewController(t)
	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSock.EXPECT().Connect().Times(1)
	mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	restore := sockhandler.SetRabbitFactory(func(string) sockhandler.SockHandler { return mockSock })
	t.Cleanup(restore)
}

// useFreshPrometheusRegistry gives the test its own default registry: requesthandler registers
// its metrics on every NewRequestHandler call and panics on a duplicate registration.
func useFreshPrometheusRegistry(t *testing.T) {
	t.Helper()

	previous := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = previous })
}

// setServiceEnv names every setting the service reads, so no value leaks in from the host
// environment or from an earlier test.
func setServiceEnv(t *testing.T, backend string, dockerAddress string, redisAddress string) {
	t.Helper()

	t.Setenv("SENTINEL_BACKEND", backend)
	t.Setenv("PROMETHEUS_ENDPOINT", fmt.Sprintf("/metrics-voip1663-%d", promEndpointSeq.Add(1)))
	t.Setenv("PROMETHEUS_LISTEN_ADDRESS", "127.0.0.1:0")
	t.Setenv("RABBITMQ_ADDRESS", "amqp://localhost:5672")
	t.Setenv("DOCKER_SOCKET_PROXY_ADDRESS", dockerAddress)
	t.Setenv("REDIS_ADDRESS", redisAddress)
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DATABASE", "1")

	// pin the kubernetes in-cluster lookup to "not in a cluster" even if the tests run in a pod.
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")
}

// closedLoopbackAddress returns a loopback address that nothing listens on.
func closedLoopbackAddress(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Could not reserve a loopback port. err: %v", err)
	}
	address := l.Addr().String()
	if errClose := l.Close(); errClose != nil {
		t.Fatalf("Could not release the loopback port. err: %v", errClose)
	}

	return address
}

// releaseSignalGoroutine hands the signal that the goroutine started by registerSignal is waiting
// for, and waits until that goroutine has read it. Only then is the shared channel torn down, so a
// leftover goroutine cannot swallow a signal sent by a later test.
func releaseSignalGoroutine(t *testing.T) {
	t.Helper()

	chSigs <- syscall.SIGTERM

	deadline := time.Now().Add(waitTimeout)
	for len(chSigs) != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("Wrong match. expect: the signal goroutine to read the signal, got: still queued")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// guardSignals makes sure the shared signal channel is empty before and after the test, and that
// the process stops relaying real signals into it once the test is over.
func guardSignals(t *testing.T) {
	t.Helper()

	drainSignals()
	t.Cleanup(func() {
		signal.Stop(chSigs)
		drainSignals()
	})
}

// loadServiceConfig loads the settings the same way run does, so runService and buildBackend see
// the values set by setServiceEnv.
func loadServiceConfig(t *testing.T) config.Config {
	t.Helper()

	if err := config.InitConfig(rootCmd); err != nil {
		t.Fatalf("Could not load the test configuration. err: %v", err)
	}

	return config.Get()
}

// runInBackground starts fn and returns the channel that receives its result.
func runInBackground(fn func() error) <-chan error {
	done := make(chan error, 1)
	go func() { done <- fn() }()

	return done
}

// waitResult returns the result of a background run or fails the test after waitTimeout.
func waitResult(t *testing.T, done <-chan error) error {
	t.Helper()

	select {
	case err := <-done:
		return err
	case <-time.After(waitTimeout):
		t.Fatalf("Wrong match. expect: the run to return, got: still running after %v", waitTimeout)
		return nil
	}
}

func Test_runService_shutsDownCleanly(t *testing.T) {
	useMockSock(t)
	useFreshPrometheusRegistry(t)

	redis := miniredis.RunT(t)
	proxy := newFakeDockerProxy(t, http.StatusOK)
	setServiceEnv(t, config.BackendDocker, proxy.address(), redis.Addr())
	loadServiceConfig(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := runInBackground(func() error { return runService(ctx, cancel) })

	// the backend is only known to be running once it has opened its event stream.
	proxy.waitEventsOpened(t)
	if got := proxy.listRequests.Load(); got != 1 {
		t.Errorf("Wrong match. expect: 1 container list at boot, got: %d", got)
	}

	cancel()

	if err := waitResult(t, done); err != nil {
		t.Errorf("Wrong match. expect: nil after a cancelled context, got: %v", err)
	}
}

func Test_runService_backendRunFailureIsReturned(t *testing.T) {
	useMockSock(t)
	useFreshPrometheusRegistry(t)

	redis := miniredis.RunT(t)
	proxy := newFakeDockerProxy(t, http.StatusInternalServerError)
	setServiceEnv(t, config.BackendDocker, proxy.address(), redis.Addr())
	loadServiceConfig(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := waitResult(t, runInBackground(func() error { return runService(ctx, cancel) }))
	if err == nil {
		t.Fatalf("Wrong match. expect: an error when the boot container list fails, got: nil")
	}
	if !strings.Contains(err.Error(), "could not run the docker monitoring backend correctly") {
		t.Errorf("Wrong match. expect: the backend run failure wrapper, got: %v", err)
	}
	if !strings.Contains(err.Error(), "could not reconcile the running containers at boot") {
		t.Errorf("Wrong match. expect: the boot reconciliation cause, got: %v", err)
	}
	if len(proxy.eventsOpened) != 0 {
		t.Errorf("Wrong match. expect: no event stream after a failed boot, got: %d", len(proxy.eventsOpened))
	}
	if ctx.Err() == nil {
		t.Errorf("Wrong match. expect: the context cancelled when runService returns, got: still alive")
	}
}

func Test_runService_backendBuildFailureIsReturned(t *testing.T) {
	tests := []struct {
		name string

		backend string
		// useReachableRedis false points the service at a closed loopback address.
		useReachableRedis bool

		expectContains []string
	}{
		{
			name: "the redis of the docker backend is unreachable",

			backend:           config.BackendDocker,
			useReachableRedis: false,

			expectContains: []string{"could not create the docker monitoring backend", "could not connect to the redis"},
		},
		{
			name: "the kubernetes backend is not running in a cluster",

			backend:           config.BackendKubernetes,
			useReachableRedis: true,

			expectContains: []string{"could not create the kubernetes monitoring backend", "could not create the in-cluster kubernetes config"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useMockSock(t)
			useFreshPrometheusRegistry(t)

			redisAddress := closedLoopbackAddress(t)
			if tt.useReachableRedis {
				redisAddress = miniredis.RunT(t).Addr()
			}
			proxy := newFakeDockerProxy(t, http.StatusOK)
			setServiceEnv(t, tt.backend, proxy.address(), redisAddress)
			loadServiceConfig(t)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			err := waitResult(t, runInBackground(func() error { return runService(ctx, cancel) }))
			if err == nil {
				t.Fatalf("Wrong match. expect: an error when the backend cannot be built, got: nil")
			}
			for _, want := range tt.expectContains {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("Wrong match. expect: %q in the error, got: %v", want, err)
				}
			}
			if proxy.listRequests.Load() != 0 {
				t.Errorf("Wrong match. expect: no docker request when the backend is not built, got: %d", proxy.listRequests.Load())
			}
		})
	}
}

func Test_buildBackend_dockerBackendIsBuilt(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	redis := miniredis.RunT(t)

	backend, cleanup, err := buildBackend(
		config.Config{
			SentinelBackend:          config.BackendDocker,
			DockerSocketProxyAddress: "tcp://127.0.0.1:1",
			RedisAddress:             redis.Addr(),
		},
		requesthandler.NewMockRequestHandler(mc),
		notifyhandler.NewMockNotifyHandler(mc),
		utilhandler.NewMockUtilHandler(mc),
	)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if backend == nil {
		t.Fatalf("Wrong match. expect: a docker backend, got: nil")
	}
	if cleanup == nil {
		t.Fatalf("Wrong match. expect: a non-nil cleanup, got: nil")
	}

	// closing the client must not fail or panic, and must be repeatable by a deferred call.
	cleanup()
}

func Test_buildBackend_dockerClientCreationFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	backend, cleanup, err := buildBackend(
		config.Config{
			SentinelBackend: config.BackendDocker,
			// no scheme: the docker client rejects the host before any connection is tried.
			DockerSocketProxyAddress: "no-scheme",
			RedisAddress:             closedLoopbackAddress(t),
		},
		requesthandler.NewMockRequestHandler(mc),
		notifyhandler.NewMockNotifyHandler(mc),
		utilhandler.NewMockUtilHandler(mc),
	)
	if cleanup == nil {
		t.Fatalf("Wrong match. expect: a non-nil cleanup, got: nil")
	}
	defer cleanup()

	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if !strings.Contains(err.Error(), "could not create the docker client") {
		t.Errorf("Wrong match. expect: the docker client creation error, got: %v", err)
	}
	if backend != nil {
		t.Errorf("Wrong match. expect: nil backend on error, got: %T", backend)
	}
}

func Test_run_startsAndStopsOnSignal(t *testing.T) {
	guardSignals(t)
	useMockSock(t)
	useFreshPrometheusRegistry(t)

	redis := miniredis.RunT(t)
	proxy := newFakeDockerProxy(t, http.StatusOK)
	setServiceEnv(t, config.BackendDocker, proxy.address(), redis.Addr())

	done := runInBackground(func() error { return run(rootCmd, nil) })

	// the signal is sent only after the backend is running, so the shutdown comes from the
	// signal and not from a context cancelled before the backend started.
	proxy.waitEventsOpened(t)
	chSigs <- syscall.SIGTERM

	if err := waitResult(t, done); err != nil {
		t.Errorf("Wrong match. expect: nil after the signal, got: %v", err)
	}
	if len(chSigs) != 0 {
		t.Errorf("Wrong match. expect: the signal goroutine to have read the signal, got: %d queued", len(chSigs))
	}
}

func Test_run_serviceFailureIsReturned(t *testing.T) {
	guardSignals(t)
	useMockSock(t)
	useFreshPrometheusRegistry(t)

	redis := miniredis.RunT(t)
	proxy := newFakeDockerProxy(t, http.StatusInternalServerError)
	setServiceEnv(t, config.BackendDocker, proxy.address(), redis.Addr())

	err := waitResult(t, runInBackground(func() error { return run(rootCmd, nil) }))

	// run registered a signal goroutine that is still waiting; release it before checking.
	releaseSignalGoroutine(t)

	if err == nil {
		t.Fatalf("Wrong match. expect: an error when the service stops with a failure, got: nil")
	}
	if !strings.Contains(err.Error(), "the sentinel-manager service stopped with an error") {
		t.Errorf("Wrong match. expect: the service failure wrapper, got: %v", err)
	}
	if !strings.Contains(err.Error(), "could not run the docker monitoring backend correctly") {
		t.Errorf("Wrong match. expect: the backend failure cause, got: %v", err)
	}
}

func Test_main_helpDoesNotRunTheService(t *testing.T) {
	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"--help"})
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})

	main()

	if !strings.Contains(out.String(), "Sentinel Manager is a microservice") {
		t.Errorf("Wrong match. expect: the long description in the help output, got: %q", out.String())
	}
}
