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
	// these four tables declare DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci; the other
	// seven tables use the database default. ai_messages.content was altered to a column level
	// charset by a later Alembic migration.
	opt := migrationcheck.Options{
		TableOptions: map[string]string{
			"ai_ai_audits":           migrationcheck.OptionUTF8MB4Collate0900AI,
			"ai_ai_prompt_histories": migrationcheck.OptionUTF8MB4Collate0900AI,
			"ai_ai_prompt_proposals": migrationcheck.OptionUTF8MB4Collate0900AI,
			"ai_aicall_participants": migrationcheck.OptionUTF8MB4Collate0900AI,
		},
		ColumnCharsets: map[string]string{
			"ai_messages.content": migrationcheck.ColumnCharsetUTF8MB4GeneralCI,
		},
	}
	if problems := migrationcheck.Check(FS, opt); len(problems) > 0 {
		t.Errorf("Wrong migration files:\n%s", strings.Join(problems, "\n"))
	}
}

func Test_baselineCreatesOwnedTables(t *testing.T) {
	up, err := fs.ReadFile(FS, "000001_baseline.up.sql")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	tests := []string{
		"ai_ai_audits", "ai_ai_prompt_histories", "ai_ai_prompt_proposals", "ai_aicall_participants",
		"ai_aicalls", "ai_ais", "ai_mcp_oauth_states", "ai_mcp_servers", "ai_messages", "ai_summaries", "ai_teams",
	}
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
