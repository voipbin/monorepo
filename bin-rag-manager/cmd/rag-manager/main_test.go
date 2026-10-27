package main

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/sirupsen/logrus"

	"monorepo/bin-rag-manager/internal/config"
	"monorepo/bin-rag-manager/pkg/raghandler"
)

// testSyntheticDSN is a PostgreSQL DSN without user information. It is never dialed because the
// migration source fails to open first.
const testSyntheticDSN = "postgres://db.example.test:5432/rag"

var testEndpointSeq atomic.Int32

// testUniqueEndpoint returns a metrics endpoint that was not registered on the default mux before.
// initProm registers it there and a second registration of the same pattern panics.
func testUniqueEndpoint() string {
	return fmt.Sprintf("/metrics-test-%d", testEndpointSeq.Add(1))
}

// testRunEnv isolates the global viper state and sets the environment the run command reads.
func testRunEnv(t *testing.T, project, region, dsn string) {
	t.Helper()

	viper.Reset()
	t.Cleanup(viper.Reset)

	t.Setenv("GCP_PROJECT_ID", project)
	t.Setenv("GCP_REGION", region)
	t.Setenv("POSTGRESQL_DSN", dsn)
	t.Setenv("PROMETHEUS_ENDPOINT", testUniqueEndpoint())
	t.Setenv("PROMETHEUS_LISTEN_ADDRESS", "127.0.0.1:0")
}

func Test_run_initConfigError(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	// a command without the flags cannot bind them
	err := run(&cobra.Command{Use: "test"}, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to initialize config") {
		t.Fatalf("Wrong match. expect error containing: failed to initialize config, got: %v", err)
	}
}

func Test_run_invalidConfiguration(t *testing.T) {
	tests := []struct {
		name string

		project string
		region  string
		dsn     string

		expectErr string
	}{
		{
			name: "project missing",

			project: "",
			region:  "test-region",
			dsn:     testSyntheticDSN,

			expectErr: "invalid configuration: GCP_PROJECT_ID is required",
		},
		{
			name: "region missing",

			project: "test-project",
			region:  "",
			dsn:     testSyntheticDSN,

			expectErr: "invalid configuration: GCP_REGION is required",
		},
		{
			name: "dsn missing",

			project: "test-project",
			region:  "test-region",
			dsn:     "",

			expectErr: "invalid configuration: POSTGRESQL_DSN is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRunEnv(t, tt.project, tt.region, tt.dsn)

			err := run(rootCmd, nil)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Fatalf("Wrong match. expect error containing: %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_run_migrationSourceError(t *testing.T) {
	testRunEnv(t, "test-project", "test-region", testSyntheticDSN)

	// no migrations directory here, so the migration source cannot be opened and no network is used
	t.Chdir(t.TempDir())

	err := run(rootCmd, nil)
	if err == nil {
		t.Fatal("Wrong match. expect an error, got: nil")
	}
	for _, expect := range []string{"could not run migrations", "failed to open source"} {
		if !strings.Contains(err.Error(), expect) {
			t.Errorf("Wrong match. expect error containing: %q, got: %v", expect, err)
		}
	}
}

func Test_runMigrations_noDSN(t *testing.T) {
	// the empty DSN is skipped before anything is opened, so the working directory does not matter
	t.Chdir(t.TempDir())

	if err := runMigrations(config.Config{}); err != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", err)
	}
}

// testHook sends the message of every matching log entry to a buffered channel without blocking.
type testHook struct {
	needle string
	ch     chan string
}

func (h *testHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *testHook) Fire(e *logrus.Entry) error {
	if !strings.Contains(e.Message, h.needle) {
		return nil
	}

	select {
	case h.ch <- e.Message:
	default:
	}

	return nil
}

// testCaptureLog installs a hook on the standard logger and restores the original hooks on cleanup.
func testCaptureLog(t *testing.T, needle string) <-chan string {
	t.Helper()

	hook := &testHook{needle: needle, ch: make(chan string, 8)}

	orig := make(logrus.LevelHooks)
	for level, hooks := range logrus.StandardLogger().Hooks {
		orig[level] = append([]logrus.Hook(nil), hooks...)
	}

	logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	logrus.StandardLogger().AddHook(hook)
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(orig) })

	return hook.ch
}

func Test_runListen(t *testing.T) {
	queue := string(commonoutline.QueueNameRagRequest)

	t.Run("consumes the request queue", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockRag := raghandler.NewMockRagHandler(mc)

		consumed := make(chan string, 1)
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, "rag-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ any, q, _ string, _, _, _ bool, _ int, _ any) error {
				consumed <- q
				return nil
			})

		if err := runListen(mockSock, mockRag); err != nil {
			t.Fatalf("Wrong match. expect: nil, got: %v", err)
		}

		select {
		case got := <-consumed:
			if got != queue {
				t.Errorf("Wrong match. expect: %v, got: %v", queue, got)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("ConsumeRPC was not called")
		}
	})

	t.Run("queue creation failure is logged and ConsumeRPC is not called", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockRag := raghandler.NewMockRagHandler(mc)

		logged := testCaptureLog(t, "Could not run the listenhandler correctly")

		// ConsumeRPC has no expectation, so a call fails the test
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(errors.New("queue failure"))

		if err := runListen(mockSock, mockRag); err != nil {
			t.Fatalf("Wrong match. expect: nil, got: %v", err)
		}

		select {
		case msg := <-logged:
			if !strings.Contains(msg, "queue failure") {
				t.Errorf("Wrong match. expect the cause in the log, got: %v", msg)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("the failure was not logged")
		}
	})
}

func Test_initProm_listenError(t *testing.T) {
	logged := testCaptureLog(t, "Could not start prometheus listener")

	// an out-of-range port fails immediately without a name lookup
	initProm(testUniqueEndpoint(), "127.0.0.1:99999")

	select {
	case <-logged:
	case <-time.After(5 * time.Second):
		t.Fatal("the listen failure was not logged")
	}
}
