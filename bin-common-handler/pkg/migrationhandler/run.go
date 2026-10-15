package migrationhandler

import "io/fs"

// Run applies the pending migrations found at the root of fsys and releases the migration
// connection. A service calls it right after its database connection, before the cache and the
// message broker.
func Run(dsn string, fsys fs.FS, versionTable string) error {
	m, err := New(dsn, fsys, versionTable)
	if err != nil {
		return err
	}
	defer func() { _ = m.Close() }()

	return m.Up()
}
