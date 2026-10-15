package migrations

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
	// Alembic declared the same table option for both tables.
	opt := migrationcheck.Options{
		TableOptions: map[string]string{
			"schedule_executions": migrationcheck.OptionUTF8MB4UnicodeCIDynamic,
			"schedule_schedules":  migrationcheck.OptionUTF8MB4UnicodeCIDynamic,
		},
	}
	if problems := migrationcheck.Check(FS, opt); len(problems) > 0 {
		t.Errorf("Wrong migration files:\n%s", strings.Join(problems, "\n"))
	}
}

// Test_seed pins the shape of the seed file: it is guarded by an empty table check, it is marked
// destructive (INSERT SELECT), it carries the nine platform schedules of the Alembic stream and
// its down does nothing.
func Test_seed(t *testing.T) {
	const upName = "000002_schedule_schedules_seed_platform_jobs.up.sql"
	const downName = "000002_schedule_schedules_seed_platform_jobs.down.sql"

	up, err := fs.ReadFile(FS, upName)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !strings.HasPrefix(string(up), "-- DESTRUCTIVE: ") {
		t.Errorf("Wrong match. %s must start with the -- DESTRUCTIVE: marker", upName)
	}
	if !strings.Contains(string(up), "WHERE NOT EXISTS (SELECT 1 FROM schedule_schedules)") {
		t.Errorf("Wrong match. %s must insert only when the table is empty", upName)
	}

	names := []string{
		"number-renew", "database-backup", "execution-retention",
		"billing-monthly-topup", "billing-failed-event-retry", "ai-proposal-expiry",
		"customer-unverified-cleanup", "customer-frozen-expiry",
		"campaign-flow-reconcile",
	}
	for _, name := range names {
		if got := strings.Count(string(up), "'"+name+"'"); got != 1 {
			t.Errorf("Wrong match. seed %s expect: 1, got: %d", name, got)
		}
	}

	down, err := fs.ReadFile(FS, downName)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if strings.Contains(strings.ToUpper(string(down)), "DELETE") {
		t.Errorf("Wrong match. %s must not delete the seeded rows", downName)
	}
}
