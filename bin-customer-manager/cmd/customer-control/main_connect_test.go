package main

import (
	"errors"
	"io"
	"strings"
	"syscall"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-customer-manager/internal/config"
)

const (
	testInvalidDSN = "invalid-dsn"
	testValidID    = "9d8e7f60-0000-4000-8000-0000000000a1"
	testValidID2   = "9d8e7f60-0000-4000-8000-0000000000a2"

	// the message every command returns when the database cannot be opened;
	// the DSN above is rejected while it is parsed, so no connection is made.
	testConnectErrPrefix = "failed to initialize handlers: could not connect to the database: database open error"
)

// useInvalidDSN makes every database connection of the commands fail while the
// DSN is parsed. The DSN is set in both viper and the loaded config so that it
// holds whichever of them is read first, and both are restored afterwards.
func useInvalidDSN(t *testing.T) {
	t.Helper()

	setViperKeys(t, map[string]any{"database_dsn": testInvalidDSN})

	old := config.Get().DatabaseDSN
	config.Get().DatabaseDSN = testInvalidDSN
	t.Cleanup(func() {
		config.Get().DatabaseDSN = old
	})
}

// Test_run_connectError covers the commands up to the first database access:
// the arguments are valid, the database cannot be opened.
func Test_run_connectError(t *testing.T) {
	tests := []struct {
		name string
		run  func(cmd *cobra.Command, args []string) error
		keys map[string]any
	}{
		{"create", runCreate, map[string]any{"email": "customer-a@example.com"}},
		{"update", runUpdate, map[string]any{"id": testValidID, "email": "customer-a@example.com"}},
		{"update billing account", runUpdateBillingAccount, map[string]any{"id": testValidID, "billing-account-id": testValidID2}},
		{"list", runList, map[string]any{"limit": 10, "token": ""}},
		{"get", runGet, map[string]any{"id": testValidID}},
		{"delete", runDelete, map[string]any{"id": testValidID}},
		{"freeze", runFreeze, map[string]any{"id": testValidID}},
		{"recover", runRecover, map[string]any{"id": testValidID}},
		{"set identity verification", runSetIdentityVerification, map[string]any{"id": testValidID, "status": "verified"}},
		{"accesskey create", runAccesskeyCreate, map[string]any{"customer-id": testValidID, "name": "key", "expire": ""}},
		{"accesskey get", runAccesskeyGet, map[string]any{"id": testValidID}},
		{"accesskey list", runAccesskeyList, map[string]any{"customer-id": testValidID, "size": 10, "token": ""}},
		{"accesskey update", runAccesskeyUpdate, map[string]any{"id": testValidID, "name": "key"}},
		{"accesskey delete", runAccesskeyDelete, map[string]any{"id": testValidID}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useInvalidDSN(t)
			setViperKeys(t, tt.keys)

			err := tt.run(&cobra.Command{}, nil)
			if err == nil || !strings.HasPrefix(err.Error(), testConnectErrPrefix) {
				t.Errorf("Wrong match. expect an error starting with: %q, got: %v", testConnectErrPrefix, err)
			}
		})
	}
}

func Test_initHandlers_connectError(t *testing.T) {
	useInvalidDSN(t)

	if _, err := initHandler(); err == nil || !strings.HasPrefix(err.Error(), "could not connect to the database: database open error") {
		t.Errorf("Wrong match. expect the database open error, got: %v", err)
	}
	if _, err := initAccesskeyHandler(); err == nil || !strings.HasPrefix(err.Error(), "could not connect to the database: database open error") {
		t.Errorf("Wrong match. expect the database open error, got: %v", err)
	}
}

func Test_initCache_connectError(t *testing.T) {
	// an address that was listening and is closed now, so the connection is refused locally
	s := miniredis.RunT(t)
	addr := s.Addr()
	s.Close()

	old := config.Get().RedisAddress
	config.Get().RedisAddress = addr
	t.Cleanup(func() {
		config.Get().RedisAddress = old
	})

	res, err := initCache()
	if !errors.Is(err, syscall.ECONNREFUSED) {
		t.Fatalf("Wrong match. expect: %v, got: %v", syscall.ECONNREFUSED, err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil handler, got: %v", res)
	}
}

// Test_initCommand_execute runs the whole command tree, so the root
// pre-run (flag binding and config loading) and the migrate command are
// executed. The database is never opened: the DSN fails to parse.
func Test_initCommand_execute(t *testing.T) {
	oldLevel := logrus.GetLevel()
	oldFormatter := logrus.StandardLogger().Formatter
	t.Cleanup(func() {
		logrus.SetLevel(oldLevel)
		logrus.SetFormatter(oldFormatter)
		// the command binds its flags to the global viper
		viper.Reset()
	})

	tests := []struct {
		name         string
		args         []string
		expectPrefix string
	}{
		{
			name:         "customer get reaches the database connection",
			args:         []string{"customer", "get", "--id", testValidID},
			expectPrefix: "failed to initialize handlers: could not connect to the database",
		},
		{
			name:         "customer get rejects a bad id before any connection",
			args:         []string{"customer", "get", "--id", "bad"},
			expectPrefix: "invalid customer ID: invalid format for Customer ID",
		},
		{
			name:         "migrate up fails while the database address is parsed",
			args:         []string{"migrate", "up"},
			expectPrefix: "could not initialize the migrator: could not parse the database dsn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useInvalidDSN(t)

			root := initCommand()
			root.SetArgs(tt.args)
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SilenceUsage = true
			root.SilenceErrors = true

			err := root.Execute()
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if !strings.HasPrefix(err.Error(), tt.expectPrefix) {
				t.Errorf("Wrong match. expect an error starting with: %q, got: %v", tt.expectPrefix, err)
			}
		})
	}
}
