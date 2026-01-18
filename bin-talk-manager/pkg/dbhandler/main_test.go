package dbhandler

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"github.com/smotes/purse"
)

var dbTest *sql.DB = nil

func TestMain(m *testing.M) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite3", `file::memory:?cache=shared`)
	if err != nil {
		logrus.Fatalf("Failed to open database: %v", err)
	}
	db.SetMaxOpenConns(1)

	// Load SQL schema files
	ps, err := purse.New(filepath.Join("../../scripts/database_scripts"))
	if err != nil {
		logrus.Fatalf("Failed to load SQL files: %v", err)
	}

	for _, file := range ps.Files() {
		contents, ok := ps.Get(file)
		if !ok {
			logrus.Fatalf("SQL file not loaded: %s", file)
		}
		_, err := db.Exec(contents)
		if err != nil {
			logrus.Fatalf("Failed to execute SQL: %v", err)
		}
	}

	dbTest = db
	defer func() {
		if err := db.Close(); err != nil {
			logrus.Errorf("Failed to close test database: %v", err)
		}
	}()

	os.Exit(m.Run())
}
