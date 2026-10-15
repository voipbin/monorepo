package mysqlmigrations

import (
	"io/fs"
	"os"
	"strings"
	"testing"

	"monorepo/bin-common-handler/pkg/migrationhandler/migrationcheck"
)

// Test_noStrayFiles fails on files that exist on disk but are not embedded (for example a
// misspelled extension), because such a file would be neither checked nor applied.
func Test_noStrayFiles(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".go") {
			continue
		}
		if _, err := fs.Stat(FS, name); err != nil {
			t.Errorf("Wrong match. %s is not embedded, migration files must end with .sql", name)
		}
	}
}

func Test_embeddedMigrations(t *testing.T) {
	// The baseline pins the character set and the collation of timeline_analyses, so the result
	// does not depend on the default of the server it runs on.
	opt := migrationcheck.Options{
		TableOptions: map[string]string{
			"timeline_analyses": migrationcheck.OptionUTF8MB4Collate0900AI,
		},
	}
	if problems := migrationcheck.Check(FS, opt); len(problems) > 0 {
		t.Errorf("Wrong migration files:\n%s", strings.Join(problems, "\n"))
	}
}

func Test_versionTable(t *testing.T) {
	if VersionTable != "timeline_schema_migrations" {
		t.Errorf("Wrong match. expect: timeline_schema_migrations, got: %s", VersionTable)
	}
}

// Test_baselineDefinition pins the columns and keys of timeline_analyses. migrationcheck does not look at
// them (the dump comparison of scripts/verify-service-migrations.sh does), and the unique key
// enforces one analysis per activeflow.
func Test_baselineDefinition(t *testing.T) {
	data, err := fs.ReadFile(FS, "000001_baseline.up.sql")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	for _, want := range []string{
		"id binary(16) NOT NULL",
		"customer_id binary(16) NOT NULL",
		"activeflow_id binary(16) NOT NULL",
		"status varchar(32) NOT NULL",
		"result json DEFAULT NULL",
		"model varchar(255) NOT NULL DEFAULT ''",
		"error text DEFAULT NULL",
		"tm_create datetime(6) NOT NULL",
		"tm_update datetime(6) DEFAULT NULL",
		"PRIMARY KEY (id)",
		"UNIQUE KEY uq_timeline_analyses_activeflow (activeflow_id)",
		"KEY idx_timeline_analyses_customer_create (customer_id, tm_create)",
	} {
		if !strings.Contains(string(data), want) {
			t.Errorf("Wrong match. the baseline does not contain: %s", want)
		}
	}
}
