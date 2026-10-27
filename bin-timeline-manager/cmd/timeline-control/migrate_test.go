package main

import (
	"strings"
	"testing"

	"monorepo/bin-timeline-manager/internal/config"
)

// setClickHouseAddress sets the process-wide address for one test and restores it afterwards.
func setClickHouseAddress(t *testing.T, address string) {
	t.Helper()

	cfg := config.Get()
	prev := cfg.ClickHouseAddress
	cfg.ClickHouseAddress = address
	t.Cleanup(func() { cfg.ClickHouseAddress = prev })
}

func Test_getMigrate_EmptyAddress(t *testing.T) {
	setClickHouseAddress(t, "")

	m, err := getMigrate()
	if err == nil || err.Error() != "CLICKHOUSE_ADDRESS is required" {
		t.Errorf("error = %v, want the required-address error", err)
	}
	if m != nil {
		t.Error("migrate instance is not nil on error")
	}
}

// With an address set, migrate.New opens the "file://migrations" source first. The package
// working directory has no migrations directory, so it fails at the source step and never
// opens a connection to the address.
func Test_getMigrate_MissingSourceDirectory(t *testing.T) {
	setClickHouseAddress(t, "192.0.2.1:9000")

	m, err := getMigrate()
	if err == nil || !strings.Contains(err.Error(), "failed to open source") {
		t.Errorf("error = %v, want a source open failure", err)
	}
	if m != nil {
		t.Error("migrate instance is not nil on error")
	}
}

func Test_migrateCommands_InitFailure(t *testing.T) {
	tests := []struct {
		name string
		run  func() error
	}{
		{"up", func() error { return cmdMigrateUp().RunE(nil, nil) }},
		{"down", func() error { return cmdMigrateDown().RunE(nil, nil) }},
		{"status", func() error { return cmdMigrateStatus().RunE(nil, nil) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setClickHouseAddress(t, "")

			err := tt.run()
			want := "failed to initialize migrate: CLICKHOUSE_ADDRESS is required"
			if err == nil || err.Error() != want {
				t.Errorf("error = %v, want %q", err, want)
			}
		})
	}
}
