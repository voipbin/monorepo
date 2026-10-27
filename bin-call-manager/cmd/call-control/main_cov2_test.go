package main

import (
	"io"
	"strings"
	"testing"

	"monorepo/bin-call-manager/internal/config"
)

// Test_initCommand_cov2_execute runs the real command tree with an unparsable database DSN, so
// every command stops at the database open step before any connection is attempted.
// The global config is loaded only once per process, so everything that needs it lives in this one test.
func Test_initCommand_cov2_execute(t *testing.T) {
	resetState(t)
	t.Setenv("DATABASE_DSN", "invalid-dsn-without-slash")

	tests := []struct {
		name string

		args []string

		expectErr string
	}{
		{
			name:      "migrate status fails to open the migrator",
			args:      []string{"migrate", "status"},
			expectErr: "could not initialize the migrator",
		},
		{
			name:      "call get fails to initialize handlers",
			args:      []string{"call", "get", "--id", "37b3a214-0afd-11eb-88ea-7bdd69288e90"},
			expectErr: "failed to initialize handlers: could not connect to the database",
		},
		{
			name:      "call list fails to initialize handlers",
			args:      []string{"call", "list", "--customer-id", "6efe9d54-7f51-11ec-95dc-4f25e8777704"},
			expectErr: "failed to initialize handlers: could not connect to the database",
		},
		{
			name:      "call delete fails to initialize handlers",
			args:      []string{"call", "delete", "--id", "37b3a214-0afd-11eb-88ea-7bdd69288e90"},
			expectErr: "failed to initialize handlers: could not connect to the database",
		},
		{
			name:      "call update-status fails to initialize handlers",
			args:      []string{"call", "update-status", "--id", "37b3a214-0afd-11eb-88ea-7bdd69288e90", "--status", "hangup"},
			expectErr: "failed to initialize handlers: could not connect to the database",
		},
		{
			name:      "call hangup fails to initialize handlers",
			args:      []string{"call", "hangup", "--id", "37b3a214-0afd-11eb-88ea-7bdd69288e90"},
			expectErr: "failed to initialize handlers: could not connect to the database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := initCommand()
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SilenceUsage = true
			root.SilenceErrors = true
			root.SetArgs(tt.args)

			err := root.Execute()
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Fatalf("Wrong match. expect error containing: %q, got: %v", tt.expectErr, err)
			}

			// the persistent pre-run hook must have loaded the global config from the environment.
			if got := config.Get().DatabaseDSN; got != "invalid-dsn-without-slash" {
				t.Errorf("Wrong match. expect: invalid-dsn-without-slash, got: %s", got)
			}
		})
	}
}
