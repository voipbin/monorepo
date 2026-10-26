package main

import (
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"monorepo/bin-tag-manager/internal/config"
)

// closedDSN returns a database address on a loopback port that nothing listens on,
// so a connection attempt is refused right away.
func closedDSN(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	return fmt.Sprintf("tcp(%s)/test", addr)
}

// injectDatabaseConfig loads the given database address into the shared config through
// config.InitConfig. LoadGlobalConfig is guarded by a sync.Once, so a flag value alone is
// ignored when the once has fired in an earlier test; the caller passes the same address
// as a flag too, which gives the same result in both cases.
func injectDatabaseConfig(t *testing.T, dsn string) {
	t.Helper()

	cmd := &cobra.Command{Use: "test"}
	for _, name := range []string{"prometheus_endpoint", "prometheus_listen_address", "rabbitmq_address", "redis_address", "redis_password"} {
		cmd.Flags().String(name, "", "")
	}
	cmd.Flags().String("database_dsn", dsn, "")
	cmd.Flags().Int("redis_database", 0, "")

	if err := config.InitConfig(cmd); err != nil {
		t.Fatalf("InitConfig returned an error: %v", err)
	}
}

func Test_commands_databaseUnreachable(t *testing.T) {
	const tagID = "11111111-2222-3333-4444-555555555555"

	tests := []struct {
		name string

		args []string

		expectErrPrefix string
	}{
		{
			name:            "create",
			args:            []string{"tag", "create", "--customer-id=" + tagID, "--name=tag-a"},
			expectErrPrefix: "failed to initialize handlers: could not connect to the database: database ping error",
		},
		{
			name:            "get",
			args:            []string{"tag", "get", "--id=" + tagID},
			expectErrPrefix: "failed to initialize handlers: could not connect to the database: database ping error",
		},
		{
			name:            "list",
			args:            []string{"tag", "list", "--customer-id=" + tagID},
			expectErrPrefix: "failed to initialize handlers: could not connect to the database: database ping error",
		},
		{
			name:            "update",
			args:            []string{"tag", "update", "--id=" + tagID, "--name=tag-b"},
			expectErrPrefix: "failed to initialize handlers: could not connect to the database: database ping error",
		},
		{
			name:            "delete",
			args:            []string{"tag", "delete", "--id=" + tagID},
			expectErrPrefix: "failed to initialize handlers: could not connect to the database: database ping error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateGlobals(t)

			dsn := closedDSN(t)
			injectDatabaseConfig(t, dsn)

			root := initCommand()
			root.SilenceUsage = true
			root.SilenceErrors = true
			root.SetOut(io.Discard)
			root.SetErr(io.Discard)
			root.SetArgs(append(tt.args, "--database_dsn="+dsn))

			err := root.Execute()
			if err == nil || !strings.HasPrefix(err.Error(), tt.expectErrPrefix) {
				t.Errorf("Wrong error. expect prefix: %s, got: %v", tt.expectErrPrefix, err)
			}
		})
	}
}

func Test_migrate_databaseUnreachable(t *testing.T) {
	isolateGlobals(t)

	dsn := closedDSN(t)
	injectDatabaseConfig(t, dsn)

	root := initCommand()
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"migrate", "status", "--database_dsn=" + dsn})

	err := root.Execute()
	if err == nil {
		t.Fatalf("Expected an error for an unreachable database, got nil")
	}
	if strings.Contains(err.Error(), "failed to initialize handlers") {
		t.Errorf("migrate must not open the handlers (Redis, RabbitMQ). got: %v", err)
	}
}
