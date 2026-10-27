package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"monorepo/bin-timeline-manager/internal/config"
)

// helperResetViper clears the global viper state (earlier tests leave overrides behind) and the
// environment variables the command tree reads, and restores a clean viper afterwards.
func helperResetViper(t *testing.T) {
	t.Helper()

	for _, key := range []string{"ID", "EVENTS", "PUBLISHER", "PAGE_TOKEN", "PAGE_SIZE", "DATABASE_DSN", "CLICKHOUSE_ADDRESS"} {
		t.Setenv(key, "")
	}

	viper.Reset()
	t.Cleanup(viper.Reset)
}

func Test_initCommand_event_list_pre_run(t *testing.T) {
	helperResetViper(t)

	cmd := initCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	// only the publisher flag is given: it reaches runEventList through the viper binding made by the
	// pre-run hook, so the first missing value reported is the resource id
	cmd.SetArgs([]string{"event", "list", "--publisher", "flow-manager"})

	err := cmd.Execute()
	if err == nil || err.Error() != "resource_id is required" {
		t.Errorf("Wrong match. expect: resource_id is required, got: %v", err)
	}
}

func Test_initCommand_migrate_mysql_status_invalid_dsn(t *testing.T) {
	helperResetViper(t)

	cmd := initCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	// the global config is loaded once on first use and overwrites its fields, so it is loaded before
	// the DSN is set; a DSN without a slash fails at the parsing step, without any network access
	config.LoadGlobalConfig()
	cfg := config.Get()
	prev := cfg.DatabaseDSN
	cfg.DatabaseDSN = "invalid-dsn-without-slash"
	t.Cleanup(func() { cfg.DatabaseDSN = prev })

	cmd.SetArgs([]string{"migrate-mysql", "status"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	for _, expect := range []string{"could not initialize the migrator", "invalid DSN"} {
		if !strings.Contains(err.Error(), expect) {
			t.Errorf("Wrong match. expect error containing: %s, got: %v", expect, err)
		}
	}
}

func Test_printJSON_unmarshalable_value(t *testing.T) {
	// a channel cannot be encoded as JSON
	err := printJSON(make(chan int))
	if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
		t.Errorf("Wrong match. expect error containing: failed to marshal JSON, got: %v", err)
	}
}
