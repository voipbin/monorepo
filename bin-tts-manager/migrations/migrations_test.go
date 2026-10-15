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
	// the Alembic stream declared DEFAULT CHARSET=utf8 for tts_manager_speaking, which is utf8mb3.
	opt := migrationcheck.Options{
		TableOptions: map[string]string{
			"tts_manager_speaking": migrationcheck.OptionUTF8MB3,
		},
	}
	if problems := migrationcheck.Check(FS, opt); len(problems) > 0 {
		t.Errorf("Wrong migration files:\n%s", strings.Join(problems, "\n"))
	}
}
