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
	// direct_directs uses the database default table options (the Alembic migration declared
	// none), so no table option is declared.
	if problems := migrationcheck.Check(FS, migrationcheck.Options{}); len(problems) > 0 {
		t.Errorf("Wrong migration files:\n%s", strings.Join(problems, "\n"))
	}
}

func Test_baselineCreatesOwnedTables(t *testing.T) {
	up, err := fs.ReadFile(FS, "000001_baseline.up.sql")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	tests := []string{"direct_directs"}
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
