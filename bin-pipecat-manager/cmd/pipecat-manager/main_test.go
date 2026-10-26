package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-pipecat-manager/internal/config"
	"monorepo/bin-pipecat-manager/pkg/pipecatcallhandler"

	"github.com/spf13/cobra"
	gomock "go.uber.org/mock/gomock"
)

// waitTimeout bounds every wait on a channel so that a regression fails the test instead of
// hanging it. It is never reached by a passing test.
const waitTimeout = 10 * time.Second

// TestMain loads the global config exactly once, because the config singleton can not be reset
// from this package. The database dsn is malformed on purpose: the driver rejects it while
// opening, so no test dials a database.
func TestMain(m *testing.M) {
	settings := map[string]string{
		"DATABASE_DSN":              "invalid-dsn",
		"REDIS_ADDRESS":             "127.0.0.1:0",
		"REDIS_PASSWORD":            "",
		"REDIS_DATABASE":            "0",
		"RABBITMQ_ADDRESS":          "",
		"PROMETHEUS_ENDPOINT":       "/metrics",
		"PROMETHEUS_LISTEN_ADDRESS": "127.0.0.1:0",
		"LOG_LEVEL":                 "info",
		"LOCAL_LLM_MODELS":          "",
	}
	for k, v := range settings {
		if err := os.Setenv(k, v); err != nil {
			panic(err)
		}
	}

	if err := config.Bootstrap(&cobra.Command{}); err != nil {
		panic(err)
	}
	config.LoadGlobalConfig()

	os.Exit(m.Run())
}

// useFreshServeMux gives the test its own default mux, so that initProm can register its endpoint
// again on every run (a second registration of one pattern panics).
func useFreshServeMux(t *testing.T) {
	t.Helper()

	orig := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	t.Cleanup(func() { http.DefaultServeMux = orig })
}

// stopSignalWatcher ends the goroutine that initSignal starts and unregisters the handler.
func stopSignalWatcher(t *testing.T) {
	t.Helper()

	t.Cleanup(func() {
		signal.Stop(chSigs)
		// the watcher goroutine ends after one signal; give it one so it does not outlive the test.
		select {
		case chSigs <- syscall.SIGTERM:
			select {
			case <-chDone:
			case <-time.After(waitTimeout):
			}
		default:
		}
	})
}

func Test_initSignal(t *testing.T) {
	t.Cleanup(func() { signal.Stop(chSigs) })

	initSignal()
	// the signal is injected through the channel the watcher reads; no process signal is sent.
	chSigs <- syscall.SIGINT

	select {
	case done := <-chDone:
		if !done {
			t.Errorf("Wrong done value. expect: true, got: false")
		}
	case <-time.After(waitTimeout):
		t.Fatalf("The watcher did not report the signal")
	}
}

func Test_initProm(t *testing.T) {
	useFreshServeMux(t)

	initProm("/metrics", "127.0.0.1:0")

	// the metrics endpoint is registered on the default mux under the given path.
	w := httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("Wrong status. expect: %d, got: %d", http.StatusOK, w.Code)
	}
	if !strings.Contains(w.Body.String(), "# HELP") {
		t.Errorf("Wrong body. expect: prometheus exposition text, got: %.80q", w.Body.String())
	}

	// any other path is not served by it.
	w = httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/other", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("Wrong status for another path. expect: %d, got: %d", http.StatusNotFound, w.Code)
	}
}

// fakeHTTPHandler is a test double of httphandler.HTTPHandler.
type fakeHTTPHandler struct {
	called chan struct{}
	err    error
}

func (f *fakeHTTPHandler) Run() error {
	close(f.called)
	return f.err
}

func Test_runHTTP(t *testing.T) {
	tests := []struct {
		name string

		runErr error
	}{
		{name: "run succeeds", runErr: nil},
		{name: "run fails and the failure is only logged", runErr: errors.New("listen failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeHTTPHandler{called: make(chan struct{}), err: tt.runErr}

			if err := runHTTP(f); err != nil {
				t.Errorf("Wrong error. expect: nil, got: %v", err)
			}

			select {
			case <-f.called:
			case <-time.After(waitTimeout):
				t.Fatalf("The http handler was not run")
			}
		})
	}
}

func Test_runListen(t *testing.T) {
	hostID := "host-a"

	t.Run("both queues are created and consumed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockPipecat := pipecatcallhandler.NewMockPipecatcallHandler(mc)

		// each queue is consumed from its own goroutine; wait until both have started so that no
		// call reaches the controller after the test has ended.
		var wg sync.WaitGroup
		wg.Add(2)

		mockSock.EXPECT().QueueCreate("bin-manager.pipecat-manager.request", "normal").Return(nil)
		mockSock.EXPECT().QueueCreate("bin-manager.pipecat-manager.request.host-a", "volatile").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), "bin-manager.pipecat-manager.request", gomock.Any(), false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ any, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ any) error {
				defer wg.Done()
				return nil
			})
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), "bin-manager.pipecat-manager.request.host-a", gomock.Any(), false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ any, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ any) error {
				defer wg.Done()
				return nil
			})

		if err := runListen(mockSock, hostID, mockPipecat); err != nil {
			t.Fatalf("Wrong error. expect: nil, got: %v", err)
		}

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(waitTimeout):
			t.Fatalf("Both queues were not consumed")
		}
	})

	t.Run("queue creation fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockPipecat := pipecatcallhandler.NewMockPipecatcallHandler(mc)

		// no consumer starts when the first queue can not be created, so the mock has no ConsumeRPC.
		mockSock.EXPECT().QueueCreate("bin-manager.pipecat-manager.request", "normal").Return(errors.New("broker down"))

		err := runListen(mockSock, hostID, mockPipecat)
		if err == nil || !strings.Contains(err.Error(), "could not run the listenhandler correctly") {
			t.Errorf("Wrong error. expect: could not run the listenhandler correctly, got: %v", err)
		}
	})
}

func Test_createDBHandler_databaseFailure(t *testing.T) {
	res, err := createDBHandler()
	if err == nil {
		t.Fatalf("Wrong error. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong result. expect: nil handler")
	}
}

func Test_run_databaseFailure(t *testing.T) {
	err := run()
	if err == nil || !strings.Contains(err.Error(), "could not create dbhandler") {
		t.Errorf("Wrong error. expect: could not create dbhandler, got: %v", err)
	}
}

func Test_runDaemon_databaseFailure(t *testing.T) {
	useFreshServeMux(t)
	stopSignalWatcher(t)

	err := runDaemon()
	if err == nil || !strings.Contains(err.Error(), "could not run the main thread") {
		t.Errorf("Wrong error. expect: could not run the main thread, got: %v", err)
	}
}
