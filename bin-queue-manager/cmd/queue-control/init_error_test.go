package main

import (
	"os"
	"strings"
	"testing"

	"monorepo/bin-queue-manager/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// useUnparsableInfra points the global config at addresses that fail while they
// are parsed, so that nothing is ever dialed, and restores the previous config
// afterwards.
func useUnparsableInfra(t *testing.T) {
	t.Helper()

	resetGlobalState(t)

	prev := *config.Get()
	*config.Get() = config.Config{
		DatabaseDSN:     "not-a-dsn",
		RedisAddress:    "no-port",
		RabbitMQAddress: "no-port",
	}
	t.Cleanup(func() { *config.Get() = prev })
}

func Test_run_initializationFails(t *testing.T) {
	validID := "6b4b2b1e-3d0c-4a6e-9b6a-0f1f2f3a4b5c"

	tests := []struct {
		name string

		run    func(*cobra.Command, []string) error
		values map[string]string

		expectInErr string
	}{
		{
			name:        "queue create after the input is validated",
			run:         runCreate,
			values:      map[string]string{"customer-id": validID, "name": "queue-a", "tag-ids": `["` + validID + `"]`},
			expectInErr: "failed to initialize handlers",
		},
		{
			name:        "queue get",
			run:         runGet,
			values:      map[string]string{"id": validID},
			expectInErr: "failed to initialize handlers",
		},
		{
			name:        "queue list",
			run:         runList,
			values:      map[string]string{"customer-id": validID},
			expectInErr: "failed to initialize handlers",
		},
		{
			name:        "queue update",
			run:         runUpdate,
			values:      map[string]string{"id": validID, "name": "queue-a"},
			expectInErr: "failed to initialize handlers",
		},
		{
			name:        "queue update tag ids",
			run:         runUpdateTagIDs,
			values:      map[string]string{"id": validID, "tag-ids": `["` + validID + `"]`},
			expectInErr: "failed to initialize handlers",
		},
		{
			name:        "queue update routing method",
			run:         runUpdateRoutingMethod,
			values:      map[string]string{"id": validID, "routing-method": "random"},
			expectInErr: "failed to initialize handlers",
		},
		{
			name:        "queue delete",
			run:         runDelete,
			values:      map[string]string{"id": validID},
			expectInErr: "failed to initialize handlers",
		},
		{
			name:        "queuecall get",
			run:         runQueuecallGet,
			values:      map[string]string{"id": validID},
			expectInErr: "failed to initialize handlers",
		},
		{
			name:        "queuecall get by reference",
			run:         runQueuecallGetByReferenceID,
			values:      map[string]string{"reference-id": validID},
			expectInErr: "failed to initialize handlers",
		},
		{
			name:        "queuecall list",
			run:         runQueuecallList,
			values:      map[string]string{"customer-id": validID},
			expectInErr: "failed to initialize handlers",
		},
		{
			name:        "queuecall delete",
			run:         runQueuecallDelete,
			values:      map[string]string{"id": validID},
			expectInErr: "failed to initialize handlers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useUnparsableInfra(t)
			for k, v := range tt.values {
				viper.Set(k, v)
			}

			out := captureStdout(t, func() {
				err := tt.run(&cobra.Command{}, nil)
				if err == nil {
					t.Fatalf("Wrong match. expect: error, got: nil")
				}

				// the failure must come from the database address being rejected
				// while it is parsed, which proves that no connection was tried.
				for _, s := range []string{tt.expectInErr, "could not connect to the database", "invalid DSN"} {
					if !strings.Contains(err.Error(), s) {
						t.Errorf("Wrong match. expect the error to contain %q, got: %v", s, err)
					}
				}
			})

			if out != "" {
				t.Errorf("Wrong match. nothing must be printed on failure, got: %q", out)
			}
		})
	}
}

func Test_initHandler_databaseFails(t *testing.T) {
	useUnparsableInfra(t)

	res, err := initHandler()
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
	if !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong match. got: %v", err)
	}
}

func Test_initQueuecallHandler_databaseFails(t *testing.T) {
	useUnparsableInfra(t)

	res, err := initQueuecallHandler()
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
	if !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong match. got: %v", err)
	}
}

func Test_initCache_connectFails(t *testing.T) {
	useUnparsableInfra(t)

	res, err := initCache()
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
	if !strings.Contains(err.Error(), "could not connect to the cache") {
		t.Errorf("Wrong match. got: %v", err)
	}
}

func Test_main_help(t *testing.T) {
	resetGlobalState(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"queue-control", "--help"}

	// --help is answered by cobra before any command runs, so main returns
	// instead of exiting.
	captureStdout(t, main)
}
