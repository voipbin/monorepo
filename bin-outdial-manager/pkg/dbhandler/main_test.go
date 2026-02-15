package dbhandler

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"

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
	ps, err := purse.New(filepath.Join("../../scripts/database_scripts"))
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

func Test_parseDestination(t *testing.T) {
	tests := []struct {
		name      string
		jsonStr   string
		expectErr bool
		expectNil bool
	}{
		{
			name:      "parse valid JSON",
			jsonStr:   `{"type":"phone","target":"+12345678900"}`,
			expectErr: false,
			expectNil: false,
		},
		{
			name:      "parse empty string",
			jsonStr:   "",
			expectErr: false,
			expectNil: true,
		},
		{
			name:      "parse invalid JSON",
			jsonStr:   `{invalid json}`,
			expectErr: true,
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dest *commonaddress.Address
			err := parseDestination(tt.jsonStr, &dest)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if tt.expectNil && dest != nil {
					t.Error("Expected nil destination but got non-nil")
				}
				if !tt.expectNil && !tt.expectErr && tt.jsonStr != "" && dest == nil {
					t.Error("Expected non-nil destination but got nil")
				}
			}
		})
	}
}

func Test_GetCurTime(t *testing.T) {
	result := GetCurTime()
	if result == "" {
		t.Error("Expected non-empty time string")
	}
	// Verify it's in the expected format (basic check)
	if len(result) < 10 {
		t.Errorf("Time string seems too short: %s", result)
	}
}
