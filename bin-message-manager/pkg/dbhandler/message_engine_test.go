//go:build dbengine

package dbhandler

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

// engineDB opens the scratch database of MESSAGE_ENGINE_DSN (MariaDB or MySQL) and creates the table of the service
// from the baseline migration. The test is skipped when the variable is not set.
func engineDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("MESSAGE_ENGINE_DSN")
	if dsn == "" {
		t.Skip("MESSAGE_ENGINE_DSN is not set")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	raw, err := os.ReadFile("../../migrations/000001_baseline.up.sql")
	if err != nil {
		t.Fatalf("read the baseline: %v", err)
	}
	for _, stmt := range strings.Split(string(raw), ";\n") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("baseline: %v", err)
		}
	}
	if _, err := db.Exec("DELETE FROM message_messages"); err != nil {
		t.Fatalf("clean: %v", err)
	}

	return db
}

func Test_engine_MessageUpdateTargetsIf(t *testing.T) {
	checkMessageUpdateTargetsIf(t, engineDB(t))
}

func Test_engine_MessageListByProviderReference(t *testing.T) {
	checkMessageListByProviderReference(t, engineDB(t))
}
