package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"monorepo/bin-outdial-manager/internal/config"
)

// unreachableDSN points at a unix socket that does not exist, so the connection fails at once
// without any database or network.
func unreachableDSN() string {
	return "test-user@unix(/nonexistent-test-dir/none.sock)/test_db"
}

// setUnreachableDatabase sets the global config to the unreachable DSN until the test ends.
func setUnreachableDatabase(t *testing.T) {
	t.Helper()

	cfg := config.Get()
	orig := *cfg
	t.Cleanup(func() { *cfg = orig })
	cfg.DatabaseDSN = unreachableDSN()
}

func Test_initHandler_databaseUnreachable(t *testing.T) {
	setUnreachableDatabase(t)

	res, err := initHandler()
	if res != nil {
		t.Errorf("expected no handler, got: %v", res)
	}
	if err == nil || !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("expected the database connection error, got: %v", err)
	}
}

func Test_run_databaseUnreachable(t *testing.T) {
	tests := []struct {
		name  string
		run   func(*cobra.Command, []string) error
		flags map[string]string
	}{
		{"create", runCreate, map[string]string{"customer-id": testCustomerID, "campaign-id": testCampaignID, "name": "test name"}},
		{"get", runGet, nil},
		{"list", runList, nil},
		{"update-basic-info", runUpdateBasicInfo, nil},
		{"update-campaign-id", runUpdateCampaignID, nil},
		{"update-data", runUpdateData, nil},
		{"delete", runDelete, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetViper(t)
			setUnreachableDatabase(t)

			for k, v := range tt.flags {
				viper.Set(k, v)
			}

			err := tt.run(&cobra.Command{}, nil)
			if err == nil {
				t.Fatalf("expected an initialization error, got nil")
			}
			for _, part := range []string{"failed to initialize handlers", "could not connect to the database"} {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("expected the error to contain %q, got: %v", part, err)
				}
			}
		})
	}
}
