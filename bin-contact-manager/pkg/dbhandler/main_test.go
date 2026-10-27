package dbhandler

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
