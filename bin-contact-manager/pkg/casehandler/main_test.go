package casehandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-redsync/redsync/v4"
	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"github.com/smotes/purse"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-contact-manager/pkg/dbhandler"
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

// Test_NewCaseHandler_wiresAllDependencies verifies NewCaseHandler's
// constructor wiring, including the VOIP-1438 redisLocker parameter --
// the returned CaseHandler must be a *caseHandler with every field
// (including peerLocks' lazy-init map and redisLocker) populated exactly
// as passed in, not silently dropped or left nil.
func Test_NewCaseHandler_wiresAllDependencies(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockLocker := NewMocklocker(mc)

	h := NewCaseHandler(mockReq, mockDB, mockNotify, mockLocker)

	ch, ok := h.(*caseHandler)
	if !ok {
		t.Fatalf("expected NewCaseHandler to return *caseHandler, got %T", h)
	}
	if ch.reqHandler != mockReq {
		t.Error("expected reqHandler to be wired to the passed-in mock")
	}
	if ch.db != mockDB {
		t.Error("expected db to be wired to the passed-in mock")
	}
	if ch.notifyHandler != mockNotify {
		t.Error("expected notifyHandler to be wired to the passed-in mock")
	}
	if ch.redisLocker != mockLocker {
		t.Error("expected redisLocker to be wired to the passed-in mock")
	}
	if ch.peerLocks == nil {
		t.Error("expected peerLocks to be initialized, got nil")
	}
	if ch.utilHandler == nil {
		t.Error("expected utilHandler to be initialized via utilhandler.NewUtilHandler(), got nil")
	}
}

// Test_NewRedsyncLocker_adaptsRealRedsync exercises the redsyncLocker
// adapter against a genuine *redsync.Redsync (no live Redis needed --
// built with zero backing pools) to cover the production wiring path
// (NewRedsyncLocker / redsyncLocker.NewMutex) that every other test in
// this package bypasses via Mocklocker. Mirrors
// bin-route-manager/pkg/healthcheckhandler's identically-named test.
func Test_NewRedsyncLocker_adaptsRealRedsync(t *testing.T) {
	rs := redsync.New() // zero pools -- never actually talks to Redis
	l := NewRedsyncLocker(rs)

	mutex := l.NewMutex("contact-manager:peerlock:test", redsync.WithTries(1))
	if mutex == nil {
		t.Fatal("Expected a non-nil mutex from NewMutex")
	}

	// Quorum (len(pools)/2+1 = 1) can never be reached with zero backing
	// pools, so this always fails -- confirming the adapter genuinely
	// round-trips into a working *redsync.Mutex rather than a stub.
	if err := mutex.LockContext(context.Background()); err == nil {
		t.Error("Expected an error acquiring a lock with zero backing pools, got nil")
	}
}
