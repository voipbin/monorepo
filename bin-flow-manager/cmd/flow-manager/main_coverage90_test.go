package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-flow-manager/internal/config"
	"monorepo/bin-flow-manager/pkg/activeflowhandler"
	"monorepo/bin-flow-manager/pkg/flowhandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

var errCov90 = errors.New("synthetic dependency failure")

// overrideConfigCov90 changes the global config for the test, and restores it when the test ends.
func overrideConfigCov90(t *testing.T, change func(cfg *config.Config)) {
	t.Helper()

	cfg := config.Get()
	prev := *cfg
	change(cfg)
	t.Cleanup(func() { *cfg = prev })
}

// Test_runDaemon_databaseFailure_coverage90 runs the daemon with a DSN that fails while it is
// parsed, so the daemon stops before any network access. The signal handler that the daemon
// installs is then checked by sending a signal value to the channel directly.
func Test_runDaemon_databaseFailure_coverage90(t *testing.T) {
	overrideConfigCov90(t, func(cfg *config.Config) {
		cfg.PrometheusEndpoint = ""
		cfg.PrometheusListenAddress = ""
		cfg.DatabaseDSN = "malformed-dsn-without-separator"
	})

	err := runDaemon()
	if err == nil || !strings.Contains(err.Error(), "could not connect to the database") {
		t.Fatalf("expected the database failure, got: %v", err)
	}

	// the signal handler installed by the daemon moves a received signal to the done channel.
	chSigs <- syscall.SIGTERM
	select {
	case done := <-chDone:
		if !done {
			t.Errorf("The done channel must receive true")
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("The signal was not handled")
	}
}

func Test_initProm_disabled_coverage90(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		listen   string
	}{
		{"the endpoint is empty", "", "127.0.0.1:0"},
		{"the listen address is empty", "/synthetic-metrics", ""},
		{"both are empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initProm(tt.endpoint, tt.listen)

			// nothing is registered on the shared mux, so a repeated run is not affected.
			if tt.endpoint == "" {
				return
			}
			req := httptest.NewRequest(http.MethodGet, tt.endpoint, nil)
			if _, pattern := http.DefaultServeMux.Handler(req); pattern != "" {
				t.Errorf("The endpoint must not be registered. pattern: %s", pattern)
			}
		})
	}
}

func Test_initCache_coverage90(t *testing.T) {
	t.Run("the cache connects to the server", func(t *testing.T) {
		server := miniredis.RunT(t)
		overrideConfigCov90(t, func(cfg *config.Config) {
			cfg.RedisAddress = server.Addr()
			cfg.RedisPassword = ""
			cfg.RedisDatabase = 0
		})

		res, err := initCache()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res == nil {
			t.Errorf("The cache handler must be returned")
		}
	})

	t.Run("the connect failure is reported", func(t *testing.T) {
		server := miniredis.RunT(t)
		addr := server.Addr()
		// the server is closed, so the address refuses the connection.
		server.Close()
		overrideConfigCov90(t, func(cfg *config.Config) {
			cfg.RedisAddress = addr
			cfg.RedisPassword = ""
			cfg.RedisDatabase = 0
		})

		res, err := initCache()
		if err == nil || !strings.Contains(err.Error(), "cache connect error") {
			t.Fatalf("expected the connect failure, got: %v", err)
		}
		if res != nil {
			t.Errorf("no cache is expected with the error. got: %v", res)
		}
	})
}

func Test_runListen_runSubscribe_coverage90(t *testing.T) {
	t.Run("runListen does not fail when the listen handler fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockSock.EXPECT().QueueCreate(gomock.Any(), "normal").Return(errCov90)

		err := runListen(mockSock, flowhandler.NewMockFlowHandler(mc), activeflowhandler.NewMockActiveflowHandler(mc), variablehandler.NewMockVariableHandler(mc))
		if err != nil {
			t.Errorf("The failure of the listen handler is only logged. got: %v", err)
		}
	})

	t.Run("runSubscribe returns the failure of the subscribe handler", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockSock.EXPECT().QueueCreate(gomock.Any(), "normal").Return(errCov90)

		err := runSubscribe(mockSock, flowhandler.NewMockFlowHandler(mc), activeflowhandler.NewMockActiveflowHandler(mc))
		if !errors.Is(err, errCov90) {
			t.Errorf("The cause must be kept. got: %v", err)
		}
	})
}
