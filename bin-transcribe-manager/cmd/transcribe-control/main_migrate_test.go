package main

import (
	"strings"
	"testing"

	"github.com/spf13/viper"

	"monorepo/bin-transcribe-manager/internal/config"
)

// Test_initCommand_migrateStatusMalformedDSN runs the migrate command through the real root
// command with a database address that cannot be parsed. The migrator is built from the config
// first, so the command must fail there, before any connection is tried.
func Test_initCommand_migrateStatusMalformedDSN(t *testing.T) {
	const malformed = "not-a-dsn"

	resetViper(t)
	saveConfig(t)

	// the configuration is loaded once per process and may be read from the environment or from
	// viper at that moment, so the malformed value is set on all of them.
	t.Setenv("DATABASE_DSN", malformed)
	viper.Set("database_dsn", malformed)
	config.Get().DatabaseDSN = malformed

	root := initCommand()
	root.SetArgs([]string{"migrate", "status"})
	root.SilenceErrors = true
	root.SilenceUsage = true

	err := root.Execute()
	if err == nil {
		t.Fatalf("Execute() expected an error")
	}
	if !strings.Contains(err.Error(), "could not parse the database dsn") {
		t.Errorf("Wrong match. expect: dsn parse error, got: %v", err)
	}
}
