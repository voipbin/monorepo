package migrations

import (
	"io/fs"
	"os"
	"regexp"
	"strings"
	"testing"

	"monorepo/bin-common-handler/pkg/migrationhandler/migrationcheck"
)

var createTableRegexp = regexp.MustCompile(`(?m)^CREATE TABLE `)

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
	// the billing tables use the database default table options, so no table option is declared.
	if problems := migrationcheck.Check(FS, migrationcheck.Options{}); len(problems) > 0 {
		t.Errorf("Wrong migration files:\n%s", strings.Join(problems, "\n"))
	}
}

func Test_baselineCreatesOwnedTables(t *testing.T) {
	up, err := fs.ReadFile(FS, "000001_baseline.up.sql")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	tests := []string{"billing_accounts", "billing_billings", "billing_failed_events"}
	for _, table := range tests {
		if !strings.Contains(string(up), "CREATE TABLE IF NOT EXISTS "+table+" (") {
			t.Errorf("Wrong match. the baseline does not create %s", table)
		}
	}
	// statements start at the beginning of a line; the header comment does not count
	if got := len(createTableRegexp.FindAllString(string(up), -1)); got != len(tests) {
		t.Errorf("Wrong match. expect: %d, got: %d", len(tests), got)
	}
}

var constraintNameRegexp = regexp.MustCompile(`(?i)\bCONSTRAINT\s+(\w+)`)

// Test_configTables checks that the two configs tables are created by their own files, one
// statement per file, that every CHECK constraint carries the prefix of its table (a CHECK
// constraint name is unique per schema, and the schema is shared with the other services), and
// that their down files carry the lossy marker.
func Test_configTables(t *testing.T) {
	tests := []struct {
		file             string
		table            string
		constraintPrefix string
		expectConstrains int
	}{
		{file: "000002_billing_configs_create_table_all", table: "billing_configs", constraintPrefix: "chk_billing_configs_", expectConstrains: 5},
		{file: "000003_billing_config_histories_create_table_all", table: "billing_config_histories", constraintPrefix: "chk_billing_config_histories_", expectConstrains: 3},
	}

	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			up, err := fs.ReadFile(FS, tt.file+".up.sql")
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !strings.HasPrefix(string(up), "CREATE TABLE IF NOT EXISTS "+tt.table+" (") {
				t.Errorf("Wrong match. the up file does not create %s", tt.table)
			}
			if got := len(createTableRegexp.FindAllString(string(up), -1)); got != 1 {
				t.Errorf("Wrong match. expect: 1 statement, got: %d", got)
			}
			if strings.Contains(string(up), "customer") {
				t.Errorf("Wrong match. the configs tables are platform wide and have no customer column")
			}

			names := constraintNameRegexp.FindAllStringSubmatch(string(up), -1)
			if len(names) != tt.expectConstrains {
				t.Errorf("Wrong match. expect: %d constraints, got: %d", tt.expectConstrains, len(names))
			}
			for _, m := range names {
				if !strings.HasPrefix(m[1], tt.constraintPrefix) {
					t.Errorf("Wrong match. the constraint %s does not start with %s", m[1], tt.constraintPrefix)
				}
			}

			down, err := fs.ReadFile(FS, tt.file+".down.sql")
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !strings.HasPrefix(string(down), "-- DOWN-LOSSY: ") || !strings.Contains(string(down), "DROP TABLE IF EXISTS "+tt.table+";") {
				t.Errorf("Wrong match. the down file is wrong: %s", down)
			}
		})
	}
}

// The migration adds the two new tables only: the baseline is untouched and no other file exists.
func Test_onlyTheTwoConfigTablesAreAdded(t *testing.T) {
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}

	expect := []string{
		"000001_baseline.down.sql", "000001_baseline.up.sql",
		"000002_billing_configs_create_table_all.down.sql", "000002_billing_configs_create_table_all.up.sql",
		"000003_billing_config_histories_create_table_all.down.sql", "000003_billing_config_histories_create_table_all.up.sql",
	}
	if strings.Join(names, ",") != strings.Join(expect, ",") {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, names)
	}
}
