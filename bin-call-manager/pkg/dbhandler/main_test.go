package dbhandler

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"github.com/smotes/purse"
)

var dbTest *sql.DB = nil // database for test

func TestMain(m *testing.M) {
	db, err := sql.Open("sqlite3", `file::memory:?cache=shared`)
	if err != nil {
		logrus.Errorf("err: %v", err)
	}
	db.SetMaxOpenConns(1)

	// Load all SQL files from specified directory into a map
	ps, err := purse.New(filepath.Join("../../scripts/database_scripts_test"))
	if err != nil {
		logrus.Infof("Err. err: %v", err)
	}
	logrus.Infof("Script loaded. scripts: %v", ps)

	for _, file := range ps.Files() {
		// Get a file's contents
		contents, ok := ps.Get(file)
		if !ok {
			logrus.Info("SQL file not loaded")
		}

		ret, err := db.Exec(contents)
		if err != nil {
			logrus.Errorf("Could not execute the sql. err: %v", err)
		}
		logrus.Infof("executed sql file. ret: %v", ret)

	}

	dbTest = db
	defer func() {
		_ = dbTest.Close()
	}()

	os.Exit(m.Run())
}

// resetTestDB deletes every row of every table in the shared test database so
// that each test starts from an empty state regardless of the tests that ran
// before it (repeated runs with -count and -shuffle).
func resetTestDB(t *testing.T) {
	t.Helper()

	rows, err := dbTest.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		t.Fatalf("could not list the test tables. err: %v", err)
	}

	tables := []string{}
	for rows.Next() {
		var name string
		if errScan := rows.Scan(&name); errScan != nil {
			_ = rows.Close()
			t.Fatalf("could not scan the table name. err: %v", errScan)
		}
		tables = append(tables, name)
	}
	errRows := rows.Err()
	// the pool has a single connection, so rows must be closed before any other statement.
	_ = rows.Close()
	if errRows != nil {
		t.Fatalf("could not read the test tables. err: %v", errRows)
	}

	for _, name := range tables {
		if _, errExec := dbTest.Exec(`DELETE FROM "` + name + `"`); errExec != nil {
			t.Fatalf("could not clear the test table. table: %s, err: %v", name, errExec)
		}
	}
}

// describeList formats every element of a result list without indexing it, so
// a failure message cannot panic when the list is empty.
func describeList[T any](list []*T) string {
	items := make([]string, 0, len(list))
	for _, item := range list {
		if item == nil {
			items = append(items, "<nil>")
			continue
		}
		items = append(items, fmt.Sprintf("%v", *item))
	}

	return "[" + strings.Join(items, ", ") + "]"
}
