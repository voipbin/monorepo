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
	defer dbTest.Close()

	os.Exit(m.Run())
}

// clean test database customers
func cleanTestDBCustomers() error {
	_, err := dbTest.Exec("delete from customers")
	if err != nil {
		return err
	}

	return nil
}
