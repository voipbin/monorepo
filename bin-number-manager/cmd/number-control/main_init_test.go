package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-number-manager/internal/config"
)

// closedPortDSN points at a loopback port without a listener, so a connection attempt is refused at once.
const closedPortDSN = "svc@tcp(127.0.0.1:1)/testdb?timeout=1s"

// closedPortAddr is a loopback address without a listener.
const closedPortAddr = "127.0.0.1:1"

// setTestConfig replaces the process wide config for one test and restores it afterwards.
// config.LoadGlobalConfig is a sync.Once, so it is consumed first: a later pre-run hook of a
// cobra command then cannot overwrite the values set here, whatever the test order is.
func setTestConfig(t *testing.T, c config.Config) {
	t.Helper()

	config.LoadGlobalConfig()
	orig := *config.Get()
	*config.Get() = c
	t.Cleanup(func() { *config.Get() = orig })
}

// restoreLog restores the logrus state that config.Bootstrap changes.
func restoreLog(t *testing.T) {
	t.Helper()

	origLevel := logrus.GetLevel()
	origFormatter := logrus.StandardLogger().Formatter
	t.Cleanup(func() {
		logrus.SetLevel(origLevel)
		logrus.SetFormatter(origFormatter)
	})
}

func Test_initHandler_databaseUnreachable(t *testing.T) {
	setTestConfig(t, config.Config{DatabaseDSN: closedPortDSN})

	res, err := initHandler()
	if err == nil || !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong match. expect error containing: could not connect to the database, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil handler, got: %v", res)
	}
}

func Test_run_initHandlerFailure(t *testing.T) {
	const validID = "5b1e1b3e-8c4a-4f7e-9d0a-2f6a3b9c7d11"

	tests := []struct {
		name string

		run  func(cmd *cobra.Command, args []string) error
		keys map[string]string
	}{
		{
			name: "create",
			run:  runCreate,
			keys: map[string]string{"customer-id": validID, "number": "+155****0001"},
		},
		{
			name: "get",
			run:  runGet,
			keys: map[string]string{"id": validID},
		},
		{
			name: "list",
			run:  runList,
			keys: map[string]string{"customer-id": validID},
		},
		{
			name: "get available",
			run:  runGetAvailable,
			keys: map[string]string{"country-code": "US"},
		},
		{
			name: "update",
			run:  runUpdate,
			keys: map[string]string{"id": validID, "name": "n1"},
		},
		{
			name: "delete",
			run:  runDelete,
			keys: map[string]string{"id": validID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			setTestConfig(t, config.Config{DatabaseDSN: closedPortDSN})
			for k, v := range tt.keys {
				viper.Set(k, v)
			}

			err := tt.run(&cobra.Command{}, nil)
			if err == nil || !strings.Contains(err.Error(), "failed to initialize handlers: could not connect to the database") {
				t.Errorf("Wrong match. expect error containing: failed to initialize handlers: could not connect to the database, got: %v", err)
			}
		})
	}
}

func Test_initCache(t *testing.T) {
	t.Run("connected", func(t *testing.T) {
		mr := miniredis.RunT(t)
		setTestConfig(t, config.Config{RedisAddress: mr.Addr()})

		res, err := initCache()
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res == nil {
			t.Errorf("Wrong match. expect: a cache handler, got: nil")
		}
	})

	t.Run("unreachable", func(t *testing.T) {
		setTestConfig(t, config.Config{RedisAddress: closedPortAddr})

		res, err := initCache()
		if err == nil || !strings.Contains(err.Error(), "could not connect to the cache") {
			t.Errorf("Wrong match. expect error containing: could not connect to the cache, got: %v", err)
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
	})
}

func Test_initCommand_migrateStatusUnreachable(t *testing.T) {
	resetViper(t)
	restoreLog(t)
	setTestConfig(t, config.Config{DatabaseDSN: closedPortDSN})

	cmd := initCommand()
	cmd.SetArgs([]string{"migrate", "status"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := cmd.Execute(); err == nil {
		t.Errorf("Wrong match. expect: an error for the unreachable database, got: nil")
	}
}

func Test_main_help(t *testing.T) {
	resetViper(t)
	restoreLog(t)

	origArgs := os.Args
	os.Args = []string{"number-control", "--help"}
	t.Cleanup(func() { os.Args = origArgs })

	out := captureStdout(t, main)

	for _, want := range []string{"Voipbin Number Management CLI", "number", "migrate"} {
		if !strings.Contains(out, want) {
			t.Errorf("Wrong match. expect the help to contain: %s, got: %s", want, out)
		}
	}
}
