package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/viper"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-route-manager/internal/config"
	"monorepo/bin-route-manager/pkg/providercallhandler"
	"monorepo/bin-route-manager/pkg/providerhandler"
	"monorepo/bin-route-manager/pkg/routehandler"
)

// unreachableDSN points at a local port that refuses the connection at once.
const unreachableDSN = "tcp(127.0.0.1:1)/test"

func TestMain(m *testing.M) {
	// the config is a singleton that is loaded only once, so the environment is prepared before the first load.
	// init() has already bound the flags and the environment variables.
	if err := os.Setenv("DATABASE_DSN", unreachableDSN); err != nil {
		panic(err)
	}
	config.LoadGlobalConfig()

	code := m.Run()
	viper.Reset()
	os.Exit(code)
}

// waitFor polls the condition until it holds or the deadline passes.
func waitFor(t *testing.T, deadline time.Duration, what string, cond func() bool) {
	t.Helper()

	end := time.Now().Add(deadline)
	for time.Now().Before(end) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("Timed out. waiting for: %s", what)
}

// Test_run covers the start of the service up to the database connection and the metrics listener.
// initProm registers the endpoint on the default mux, so every run call needs its own endpoint.
func Test_run(t *testing.T) {
	// occupy the listen address so that the first attempt of the metrics listener fails
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}
	defer func() { _ = occupied.Close() }()
	addr := occupied.Addr().String()

	cfg := config.Get()
	orig := cfg.PrometheusListenAddress
	t.Cleanup(func() { cfg.PrometheusListenAddress = orig })
	cfg.PrometheusListenAddress = addr
	// every run registers its endpoint on the default mux, so the endpoint is unique per call
	origEndpoint := cfg.PrometheusEndpoint
	t.Cleanup(func() { cfg.PrometheusEndpoint = origEndpoint })
	cfg.PrometheusEndpoint = uniqueEndpoint()

	hook := logrustest.NewGlobal()
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks)) })

	runErr := run(rootCmd, nil)
	if runErr == nil || !strings.Contains(runErr.Error(), "database ping error") {
		t.Fatalf("Wrong match. expect: database ping error, got: %v", runErr)
	}

	// the endpoint is registered on the default mux
	_, pattern := http.DefaultServeMux.Handler(httptest.NewRequest(http.MethodGet, cfg.PrometheusEndpoint, nil))
	if pattern != cfg.PrometheusEndpoint {
		t.Errorf("Wrong match. expect: %s registered, got: %q", cfg.PrometheusEndpoint, pattern)
	}

	// the occupied address makes the listener fail and retry
	waitFor(t, 5*time.Second, "the listener failure log", func() bool {
		for _, e := range hook.AllEntries() {
			if strings.Contains(e.Message, "Could not start prometheus listener") {
				return true
			}
		}
		return false
	})

	// once the address is free the retry succeeds and the metrics are served
	_ = occupied.Close()
	client := &http.Client{Timeout: time.Second}
	waitFor(t, 10*time.Second, "the metrics server", func() bool {
		res, errGet := client.Get("http://" + addr + cfg.PrometheusEndpoint)
		if errGet != nil {
			return false
		}
		defer func() { _ = res.Body.Close() }()
		return res.StatusCode == http.StatusOK
	})
}

func Test_runRequestListen(t *testing.T) {
	queue := string(commonoutline.QueueNameRouteRequest)

	t.Run("starts consuming the request queue", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockProvider := providerhandler.NewMockProviderHandler(mc)
		mockProviderCall := providercallhandler.NewMockProviderCallHandler(mc)
		mockRoute := routehandler.NewMockRouteHandler(mc)

		// the listen handler consumes in a goroutine, so wait for the call
		consuming := make(chan struct{})
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, "route-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(context.Context, string, string, bool, bool, bool, int, sock.CbMsgRPC) error {
				close(consuming)
				return nil
			})

		if err := runRequestListen(mockSock, mockProvider, mockProviderCall, mockRoute); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		select {
		case <-consuming:
		case <-time.After(5 * time.Second):
			t.Fatalf("Timed out. waiting for ConsumeRPC")
		}
	})

	t.Run("a queue failure is logged and does not start consuming", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockProvider := providerhandler.NewMockProviderHandler(mc)
		mockProviderCall := providercallhandler.NewMockProviderCallHandler(mc)
		mockRoute := routehandler.NewMockRouteHandler(mc)

		// ConsumeRPC has no expectation: a call would fail the test
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(context.DeadlineExceeded)

		if err := runRequestListen(mockSock, mockProvider, mockProviderCall, mockRoute); err != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", err)
		}
	})
}
