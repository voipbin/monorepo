package dbhandler

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

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
		logrus.Infof("executed sql file. filename: %s, ret: %v", file, ret)

	}

	dbTest = db
	defer func() {
		_ = dbTest.Close()
	}()

	os.Exit(m.Run())
}

// resetTestTables removes every row of the tables of the shared test database,
// so that each test starts from an empty state regardless of the run order or count.
func resetTestTables(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := dbTest.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		t.Fatalf("Could not list the test tables. err: %v", err)
	}

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			_ = rows.Close()
			t.Fatalf("Could not read the table name. err: %v", err)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		t.Fatalf("Could not iterate the test tables. err: %v", err)
	}
	_ = rows.Close()

	for _, name := range tables {
		if _, err := dbTest.ExecContext(ctx, "DELETE FROM "+name); err != nil {
			t.Fatalf("Could not clear the test table. table: %s, err: %v", name, err)
		}
	}
}
