package dbhandler

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

var dbTest *sql.DB // in-memory sqlite database of the config store tests

// TestMain loads the sqlite schema of the configs tables into one shared in-memory database. The
// other tests of this package use sqlmock and do not touch it.
func TestMain(m *testing.M) {
	db, err := sql.Open("sqlite3", `file::memory:?cache=shared`)
	if err != nil {
		logrus.Errorf("Could not open the test database. err: %v", err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(1)

	files, err := filepath.Glob(filepath.Join("..", "..", "scripts", "database_scripts_test", "*.sql"))
	if err != nil || len(files) == 0 {
		logrus.Errorf("Could not find the test schema files. err: %v", err)
		os.Exit(1)
	}
	for _, f := range files {
		contents, errRead := os.ReadFile(f)
		if errRead != nil {
			logrus.Errorf("Could not read the schema file. file: %s, err: %v", f, errRead)
			os.Exit(1)
		}
		if _, errExec := db.Exec(string(contents)); errExec != nil {
			logrus.Errorf("Could not execute the schema file. file: %s, err: %v", f, errExec)
			os.Exit(1)
		}
	}

	dbTest = db
	code := m.Run()
	_ = dbTest.Close()

	os.Exit(code)
}
