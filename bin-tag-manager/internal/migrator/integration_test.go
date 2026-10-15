package migrator

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"
)

// The tests in this file need a real MariaDB or MySQL server. They run only when
// MIGRATOR_TEST_DSN points at a throwaway database (scripts/verify-service-migrations.sh sets it).

func integrationDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("MIGRATOR_TEST_DSN")
	if dsn == "" {
		t.Skip("MIGRATOR_TEST_DSN is not set")
	}
	return dsn
}

// integrationFS builds migrations for a uniquely named table so that tests do not collide.
func integrationFS(table string, second string) fstest.MapFS {
	res := fstest.MapFS{
		"000001_create.up.sql":   {Data: []byte(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id int NOT NULL, PRIMARY KEY (id));", table))},
		"000001_create.down.sql": {Data: []byte("SELECT 1;")},
	}
	if second != "" {
		res["000002_alter.up.sql"] = &fstest.MapFile{Data: []byte(second)}
		res["000002_alter.down.sql"] = &fstest.MapFile{Data: []byte("SELECT 1;")}
	}
	return res
}

func integrationExec(t *testing.T, dsn string, query string) {
	t.Helper()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("could not open the database: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(query); err != nil {
		t.Fatalf("could not execute %q: %v", query, err)
	}
}

func integrationCleanup(t *testing.T, dsn string, tables ...string) {
	t.Helper()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("could not open the database: %v", err)
	}
	defer func() { _ = db.Close() }()

	for _, table := range tables {
		if _, err := db.Exec("DROP TABLE IF EXISTS " + table); err != nil {
			t.Fatalf("could not drop %s: %v", table, err)
		}
	}
}

func Test_integration_upDownUp(t *testing.T) {
	dsn := integrationDSN(t)
	table, versionTable := "it_updownup", "it_updownup_schema_migrations"
	integrationCleanup(t, dsn, table, versionTable)
	defer integrationCleanup(t, dsn, table, versionTable)

	fsys := integrationFS(table, "ALTER TABLE it_updownup ADD COLUMN c int NULL;")
	fsys["000002_alter.down.sql"] = &fstest.MapFile{Data: []byte("ALTER TABLE it_updownup DROP COLUMN c;")}

	m, err := New(dsn, fsys, versionTable)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	defer func() { _ = m.Close() }()

	steps := []struct {
		name   string
		run    func() error
		expect uint
	}{
		{"up", m.Up, 2},
		{"up again is no change", m.Up, 2},
		{"down", m.Down, 1},
		{"up", m.Up, 2},
	}
	for _, s := range steps {
		if err := s.run(); err != nil {
			t.Fatalf("%s: Wrong match. expect: ok, got: %v", s.name, err)
		}
		version, dirty, err := m.Status()
		if err != nil || dirty || version != s.expect {
			t.Fatalf("%s: Wrong match. expect: %d clean, got: %d dirty=%v err=%v", s.name, s.expect, version, dirty, err)
		}
	}
}

func Test_integration_aheadOfImage(t *testing.T) {
	dsn := integrationDSN(t)
	table, versionTable := "it_ahead", "it_ahead_schema_migrations"
	integrationCleanup(t, dsn, table, versionTable)
	defer integrationCleanup(t, dsn, table, versionTable)

	m, err := New(dsn, integrationFS(table, ""), versionTable)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	defer func() { _ = m.Close() }()

	if err := m.Up(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// A newer image migrated to version 2 and this older image restarts.
	if err := m.Force(2); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if err := m.Up(); err != nil {
		t.Fatalf("Wrong match. expect: skip without error, got: %v", err)
	}
	if version, dirty, _ := m.Status(); version != 2 || dirty {
		t.Errorf("Wrong match. expect: 2 clean, got: %d dirty=%v", version, dirty)
	}

	// A dirty database is refused even when it is ahead.
	integrationExec(t, dsn, "UPDATE "+versionTable+" SET dirty = 1")
	if err := m.Up(); err == nil || !strings.Contains(err.Error(), "Dirty database version 2") {
		t.Errorf("Wrong match. expect: dirty refusal, got: %v", err)
	}
}

func Test_integration_dirtyAndForce(t *testing.T) {
	dsn := integrationDSN(t)
	table, versionTable := "it_dirty", "it_dirty_schema_migrations"
	integrationCleanup(t, dsn, table, versionTable)
	defer integrationCleanup(t, dsn, table, versionTable)

	m, err := New(dsn, integrationFS(table, "ALTER TABLE it_dirty ADD COLUMN broken SYNTAX ERROR;"), versionTable)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	defer func() { _ = m.Close() }()

	if err := m.Up(); err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if version, dirty, _ := m.Status(); version != 2 || !dirty {
		t.Fatalf("Wrong match. expect: 2 dirty, got: %d dirty=%v", version, dirty)
	}
	if err := m.Up(); err == nil || !strings.Contains(err.Error(), "Dirty database version 2") {
		t.Fatalf("Wrong match. expect: dirty refusal, got: %v", err)
	}

	if err := m.Force(1); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if version, dirty, _ := m.Status(); version != 1 || dirty {
		t.Errorf("Wrong match. expect: 1 clean, got: %d dirty=%v", version, dirty)
	}

	// version 0 is refused: it would make the next start fail.
	if err := m.Force(0); err == nil {
		t.Errorf("Wrong match. expect: error for version 0, got: nil")
	}

	// -1 resets the version table to the unversioned state.
	if err := m.Force(-1); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if version, dirty, err := m.Status(); err != nil || version != 0 || dirty {
		t.Errorf("Wrong match. expect: 0 clean, got: %d dirty=%v err=%v", version, dirty, err)
	}

	// down on an unversioned database is a no-op.
	if err := m.Down(); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_integration_concurrentUp(t *testing.T) {
	dsn := integrationDSN(t)
	table, versionTable := "it_concurrent", "it_concurrent_schema_migrations"
	integrationCleanup(t, dsn, table, versionTable)
	defer integrationCleanup(t, dsn, table, versionTable)

	fsys := integrationFS(table, "")

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m, err := New(dsn, fsys, versionTable)
			if err != nil {
				errs[i] = err
				return
			}
			defer func() { _ = m.Close() }()
			errs[i] = m.Up()
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("Wrong match. process %d expect: ok, got: %v", i, err)
		}
	}

	m, err := New(dsn, fsys, versionTable)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	defer func() { _ = m.Close() }()
	if version, dirty, _ := m.Status(); version != 1 || dirty {
		t.Errorf("Wrong match. expect: 1 clean, got: %d dirty=%v", version, dirty)
	}
}

// Test_integration_lockWaitTimeout holds a read transaction on the table (a metadata lock) and
// checks that an ALTER migration fails after lock_wait_timeout instead of waiting forever, and
// that the dirty state is recoverable with Force(N-1).
func Test_integration_lockWaitTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("takes about lock_wait_timeout seconds")
	}
	dsn := integrationDSN(t)
	table, versionTable := "it_lock", "it_lock_schema_migrations"
	integrationCleanup(t, dsn, table, versionTable)
	defer integrationCleanup(t, dsn, table, versionTable)

	// version 1 first, so that only the ALTER is pending.
	first, err := New(dsn, integrationFS(table, ""), versionTable)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if err := first.Up(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	_ = first.Close()

	holder, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("could not open the database: %v", err)
	}
	defer func() { _ = holder.Close() }()
	tx, err := holder.Begin()
	if err != nil {
		t.Fatalf("could not begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.Query("SELECT * FROM " + table)
	if err != nil {
		t.Fatalf("could not read: %v", err)
	}
	_ = rows.Close()

	m, err := New(dsn, integrationFS(table, "ALTER TABLE it_lock ADD COLUMN c int NULL;"), versionTable)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	defer func() { _ = m.Close() }()

	start := time.Now()
	err = m.Up()
	elapsed := time.Since(start)
	if err == nil || !strings.Contains(err.Error(), "1205") {
		t.Fatalf("Wrong match. expect: Error 1205 lock wait timeout, got: %v", err)
	}
	if elapsed < time.Duration(lockWaitTimeoutSeconds-1)*time.Second || elapsed > time.Duration(lockWaitTimeoutSeconds+10)*time.Second {
		t.Errorf("Wrong match. expect: about %ds, got: %v", lockWaitTimeoutSeconds, elapsed)
	}
	if version, dirty, _ := m.Status(); version != 2 || !dirty {
		t.Errorf("Wrong match. expect: 2 dirty, got: %d dirty=%v", version, dirty)
	}

	_ = tx.Rollback()
	if err := m.Force(1); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if err := m.Up(); err != nil {
		t.Fatalf("Wrong match. expect: retry ok, got: %v", err)
	}
	if version, dirty, _ := m.Status(); version != 2 || dirty {
		t.Errorf("Wrong match. expect: 2 clean, got: %d dirty=%v", version, dirty)
	}
}
