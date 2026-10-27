// Package migrationhandler applies the embedded schema migrations of a service that owns its
// tables with golang-migrate, and provides the recovery command (status, up, down, force) that
// the service control binary registers.
package migrationhandler

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"sync/atomic"

	"github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	dbmysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/sirupsen/logrus"
)

// lockWaitTimeoutSeconds bounds how long a migration statement waits for a metadata lock.
// Without it an ALTER waits as long as the server default (up to a year) and blocks every
// following query on the table.
const lockWaitTimeoutSeconds = 10

// Migrator applies the migrations through a dedicated connection pool, separate from the
// service pool.
type Migrator struct {
	m          Engine
	maxVersion uint
}

// New opens a dedicated database connection from the service DSN and prepares the migrations
// found at the root of fsys. versionTable is the table that records the applied version.
//
// The DSN is parsed by the go-sql-driver and never re-encoded into a URL, so passwords that
// contain characters such as % or + are used as they are.
func New(dsn string, fsys fs.FS, versionTable string) (*Migrator, error) {
	e, maxVersion, err := (*engineFunc.Load())(dsn, fsys, versionTable)
	if err != nil {
		return nil, err
	}

	return &Migrator{m: e, maxVersion: maxVersion}, nil
}

// Engine is the part of the golang-migrate runner that Migrator uses. Close releases both the
// migration source and the database connection and returns their errors separately. It is exported
// so that a test can provide a fake through SetEngineFunc.
type Engine interface {
	Version() (version uint, dirty bool, err error)
	Up() error
	Steps(n int) error
	Force(version int) error
	Close() (source error, database error)
}

// engineFunc creates the engine and returns the highest migration version found. Tests replace it
// with SetEngineFunc.
var engineFunc atomic.Pointer[func(dsn string, fsys fs.FS, versionTable string) (Engine, uint, error)]

func init() {
	SetEngineFunc(nil)
}

// SetEngineFunc replaces the function that New uses to open the engine and returns a function that
// puts the previous one back. A nil f installs the default. The function returns the engine and the
// highest migration version; the engine must not be nil when the error is nil.
// It is meant for tests: call the returned function with defer or t.Cleanup, in reverse order of
// the calls, and do not run such a test in parallel with a test that expects the default.
func SetEngineFunc(f func(dsn string, fsys fs.FS, versionTable string) (Engine, uint, error)) (restore func()) {
	if f == nil {
		f = defaultEngine
	}

	prev := engineFunc.Swap(&f)
	return func() {
		engineFunc.Store(prev)
	}
}

// defaultEngine opens a dedicated database connection from the service DSN and prepares the
// migrations found at the root of fsys. Every error path returns an untyped nil engine.
func defaultEngine(dsn string, fsys fs.FS, versionTable string) (Engine, uint, error) {
	cfg, err := newConfig(dsn)
	if err != nil {
		return nil, 0, err
	}

	src, maxVersion, err := newSource(fsys)
	if err != nil {
		return nil, 0, err
	}

	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		_ = src.Close()
		return nil, 0, fmt.Errorf("could not create the database connector: %w", err)
	}
	db := sql.OpenDB(connector)

	drv, err := dbmysql.WithInstance(db, &dbmysql.Config{
		MigrationsTable: versionTable,
		DatabaseName:    cfg.DBName,
	})
	if err != nil {
		_ = src.Close()
		_ = db.Close()
		return nil, 0, fmt.Errorf("could not initialize the migration database driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "mysql", drv)
	if err != nil {
		_ = src.Close()
		_ = db.Close()
		return nil, 0, fmt.Errorf("could not initialize migrate: %w", err)
	}

	return m, maxVersion, nil
}

// newConfig derives the migration connection settings from the service DSN.
func newConfig(dsn string) (*mysql.Config, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("could not parse the database dsn: %w", err)
	}

	// Required by the driver for files that hold several statements (the baseline does).
	cfg.MultiStatements = true
	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}
	cfg.Params["lock_wait_timeout"] = strconv.Itoa(lockWaitTimeoutSeconds)

	return cfg, nil
}

// newSource opens the migration files and returns the highest version found. Every error is
// returned: a duplicate version number is reported here instead of failing later.
func newSource(fsys fs.FS) (source.Driver, uint, error) {
	src, err := iofs.New(fsys, ".")
	if err != nil {
		return nil, 0, fmt.Errorf("could not read the migration files: %w", err)
	}

	version, err := src.First()
	if err != nil {
		_ = src.Close()
		return nil, 0, fmt.Errorf("could not find the first migration: %w", err)
	}

	for {
		next, errNext := src.Next(version)
		if errors.Is(errNext, os.ErrNotExist) {
			break
		}
		if errNext != nil {
			_ = src.Close()
			return nil, 0, fmt.Errorf("could not walk the migration files: %w", errNext)
		}
		version = next
	}

	return src, version, nil
}

// isAhead reports whether the database is at a clean version newer than any migration of this
// image. A dirty database is never treated as ahead.
func isAhead(current uint, dirty bool, maxVersion uint) bool {
	return !dirty && current > maxVersion
}

// Up applies all pending migrations.
//
// When the database is already ahead of this image (for example an older container restarts
// after a newer image migrated), the migrations are skipped with a warning instead of failing,
// so that an image rollback keeps working. A dirty database is refused.
func (h *Migrator) Up() error {
	log := logrus.WithField("func", "Up")

	current, dirty, err := h.m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("could not get the migration version: %w", err)
	}

	if err == nil && isAhead(current, dirty, h.maxVersion) {
		log.Warnf("db version %d is ahead of this image (max %d); skipping migrations", current, h.maxVersion)
		return nil
	}

	errUp := h.m.Up()
	if errors.Is(errUp, migrate.ErrNoChange) {
		log.Infof("no pending migrations. version: %d", current)
		return nil
	}
	if errUp != nil {
		return fmt.Errorf("could not apply the migrations: %w", errUp)
	}

	log.Infof("migrations applied. from: %d, to: %d", current, h.maxVersion)
	return nil
}

// Down reverts the latest applied migration.
func (h *Migrator) Down() error {
	// Nothing is applied, so there is nothing to revert (Steps would fail with a missing file).
	if version, _, err := h.Status(); err != nil {
		return err
	} else if version == 0 {
		return nil
	}

	if err := h.m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("could not revert the migration: %w", err)
	}
	return nil
}

// Status returns the applied version (0 when nothing is applied) and the dirty flag.
func (h *Migrator) Status() (version uint, dirty bool, err error) {
	version, dirty, err = h.m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("could not get the migration version: %w", err)
	}
	return version, dirty, nil
}

// MaxVersion returns the highest migration version embedded in this image.
func (h *Migrator) MaxVersion() uint {
	return h.maxVersion
}

// Force records the given version as applied and clears the dirty flag without running any
// migration. -1 resets the version table to the unversioned state.
func (h *Migrator) Force(version int) error {
	// Version 0 cannot be represented: the next start would fail with "no migration found for
	// version 0". The unversioned state is -1.
	if version == 0 {
		return fmt.Errorf("version 0 is not allowed, use -1 to reset to the unversioned state")
	}
	if version > 0 && uint(version) > h.maxVersion {
		logrus.WithField("func", "Force").Warnf("forcing version %d which is ahead of this image (max %d)", version, h.maxVersion)
	}

	if err := h.m.Force(version); err != nil {
		return fmt.Errorf("could not force the migration version: %w", err)
	}
	return nil
}

// Close releases the migration connection (the pool opened by New is closed with it).
func (h *Migrator) Close() error {
	srcErr, dbErr := h.m.Close()
	return errors.Join(srcErr, dbErr)
}
