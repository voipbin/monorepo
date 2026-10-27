package dbhandler

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/smotes/purse"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/pkg/cachehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// newPrivateDB opens a private in-memory sqlite database with the test schema.
// Unlike the package handle it is owned by the calling test, so a test may fill
// it with unusual rows or close it without affecting any other test.
func newPrivateDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("could not open the private database. err: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	ps, err := purse.New(filepath.Join("../../scripts/database_scripts_test"))
	if err != nil {
		t.Fatalf("could not load the schema scripts. err: %v", err)
	}
	for _, file := range ps.Files() {
		contents, ok := ps.Get(file)
		if !ok {
			t.Fatalf("schema script %s was not loaded", file)
		}
		if _, errExec := db.Exec(contents); errExec != nil {
			t.Fatalf("could not apply schema script %s. err: %v", file, errExec)
		}
	}

	return db
}

// newClosedDB returns a private database that is already closed, so that every
// statement run on it fails.
func newClosedDB(t *testing.T) *sql.DB {
	t.Helper()

	db := newPrivateDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("could not close the private database. err: %v", err)
	}

	return db
}

// newCov90Handler builds a handler on the given database with mocked time and cache.
func newCov90Handler(t *testing.T, db *sql.DB) (*handler, *utilhandler.MockUtilHandler, *cachehandler.MockCacheHandler) {
	t.Helper()

	mc := gomock.NewController(t)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil.EXPECT().TimeNow().Return(nil).AnyTimes()
	mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000").AnyTimes()

	return &handler{utilHandler: mockUtil, db: db, cache: mockCache}, mockUtil, mockCache
}
