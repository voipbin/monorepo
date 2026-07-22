package casehandler

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"github.com/smotes/purse"
)

var dbTest *sql.DB = nil // database for test

// sqliteJSONExtract is a minimal stand-in for MySQL's
// JSON_UNQUOTE(JSON_EXTRACT(col, '$.field')), used by the STORED
// generated columns in scripts/database_scripts_test/contacts.sql
// (peer_type/peer_target/local_type/local_target derived from the
// peer/local JSON columns). The vendored github.com/mattn/go-sqlite3
// build here does NOT compile in SQLite's json1 extension (no
// sqlite_json build tag anywhere in this monorepo's go test
// invocation), so plain `json_extract(...)` is unavailable; this
// registers an equivalent scalar function on the driver instead. Only
// supports the single-level "$.key" paths actually used by this
// design -- not a general JSONPath implementation. Must return a
// concrete, driver-supported type (string/[]byte/bool/int/float; see
// go-sqlite3's callbackRet): sql.NullString and interface{} both
// panic/error via reflection here, so a missing key or NULL input
// simply maps to "" rather than SQL NULL -- adequate for this test
// schema, where no test asserts NULL-ness of the derived
// local_type/local_target columns (only their string value).
func sqliteJSONExtract(js, path string) (string, error) {
	if js == "" {
		return "", nil
	}
	key := path
	if len(path) > 2 && path[:2] == "$." {
		key = path[2:]
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(js), &m); err != nil {
		return "", nil //nolint:nilerr // malformed/non-object JSON -> "", matches JSON_EXTRACT's leniency
	}
	v, ok := m[key]
	if !ok || v == nil {
		return "", nil
	}
	if s, ok := v.(string); ok {
		return s, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func init() {
	sql.Register("sqlite3_test", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			return conn.RegisterFunc("json_extract", sqliteJSONExtract, true)
		},
	})
}

func TestMain(m *testing.M) {
	db, err := sql.Open("sqlite3_test", `file::memory:?cache=shared`)
	if err != nil {
		logrus.Errorf("err: %v", err)
	}
	db.SetMaxOpenConns(1)

	ps, err := purse.New(filepath.Join("../../scripts/database_scripts_test"))
	if err != nil {
		logrus.Infof("Err. err: %v", err)
	}

	for _, file := range ps.Files() {
		contents, ok := ps.Get(file)
		if !ok {
			logrus.Info("SQL file not loaded")
		}

		if _, err := db.Exec(contents); err != nil {
			logrus.Errorf("Could not execute the sql. err: %v", err)
		}
	}

	dbTest = db
	defer func() {
		_ = dbTest.Close()
	}()

	os.Exit(m.Run())
}
