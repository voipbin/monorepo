package main

import (
	"net"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-transcribe-manager/internal/config"
)

// saveConfig restores the whole process-wide config once the test is over.
func saveConfig(t *testing.T) {
	t.Helper()

	saved := *config.Get()
	t.Cleanup(func() { *config.Get() = saved })
}

// closedAddr returns a loopback address that nothing listens on.
func closedAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	return addr
}

func Test_initCache(t *testing.T) {
	saveConfig(t)

	t.Run("connects to a reachable server", func(t *testing.T) {
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("could not start miniredis. err: %v", err)
		}
		t.Cleanup(mr.Close)

		config.Get().RedisAddress = mr.Addr()

		res, err := initCache()
		if err != nil {
			t.Fatalf("initCache() returned an error. err: %v", err)
		}
		if res == nil {
			t.Fatalf("initCache() returned a nil handler")
		}
	})

	t.Run("wraps the connect error", func(t *testing.T) {
		config.Get().RedisAddress = closedAddr(t)

		res, err := initCache()
		if err == nil {
			t.Fatalf("initCache() expected an error")
		}
		if res != nil {
			t.Errorf("initCache() expected a nil handler on error. got: %v", res)
		}
		if !strings.Contains(err.Error(), "could not connect to the cache") {
			t.Errorf("initCache() error was not wrapped. err: %v", err)
		}
	})
}

// closedDBConfig points the database at a loopback port nothing listens on, so initHandler
// fails at the database step and never reaches the cache or RabbitMQ.
func closedDBConfig(t *testing.T) {
	t.Helper()

	saveConfig(t)
	config.Get().DatabaseDSN = "tester@tcp(" + closedAddr(t) + ")/transcribe_test"
}

func Test_initHandler_databaseError(t *testing.T) {
	closedDBConfig(t)

	res, err := initHandler()
	if err == nil {
		t.Fatalf("initHandler() expected an error")
	}
	if res != nil {
		t.Errorf("initHandler() expected a nil handler on error. got: %v", res)
	}
	if !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("initHandler() error was not wrapped. err: %v", err)
	}
}

// Test_run_initHandlerError checks that every run function reports a handler initialization
// failure instead of continuing, and that the valid-input commands reach that step.
func Test_run_initHandlerError(t *testing.T) {
	const id = "b25a1e10-0000-11ee-9c11-000000000001"

	tests := []struct {
		name string
		run  func(*cobra.Command, []string) error
		set  map[string]string
	}{
		{"stop", runStop, map[string]string{"id": id}},
		{"get", runGet, map[string]string{"id": id}},
		{"get by reference id and language", runGetByReferenceIDAndLanguage, map[string]string{
			"customer-id": id, "reference-id": id, "language": "en-US",
		}},
		{"list", runList, map[string]string{"customer-id": id}},
		{"delete", runDelete, map[string]string{"id": id}},
		{"start", runStart, map[string]string{
			"customer-id": id, "activeflow-id": id, "on-end-flow-id": id, "reference-id": id,
			"reference-type": "call",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			closedDBConfig(t)
			for k, v := range tt.set {
				viper.Set(k, v)
			}

			err := tt.run(&cobra.Command{}, nil)
			if err == nil {
				t.Fatalf("run expected an error")
			}
			if !strings.Contains(err.Error(), "failed to initialize handlers") {
				t.Errorf("run error is not the handler initialization error. err: %v", err)
			}
		})
	}
}

// Test_initCommand_persistentPreRun executes a real command through the root command, so the
// PersistentPreRunE hook runs. It must bind the flags of the executed command to viper and must
// not fail. LoadGlobalConfig also runs there, but it loads only once per process, so the loaded
// values are not asserted here (internal/config covers them).
func Test_initCommand_persistentPreRun(t *testing.T) {
	resetViper(t)
	saveConfig(t)

	root := initCommand()
	ran := false
	noop := &cobra.Command{
		Use: "noop",
		RunE: func(cmd *cobra.Command, args []string) error {
			ran = true
			return nil
		},
	}
	noop.Flags().String("probe", "default-value", "probe flag")
	root.AddCommand(noop)
	root.SetArgs([]string{"noop", "--probe=from-flag"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() returned an error. err: %v", err)
	}
	if !ran {
		t.Errorf("the subcommand did not run")
	}
	if got := viper.GetString("probe"); got != "from-flag" {
		t.Errorf("the flag was not bound to viper. expect: from-flag, got: %s", got)
	}
}
