// Package migrations embeds the schema migration files of bin-queue-manager.
//
// File format: NNNNNN_<table>_<verb>_<type>_<items>.up.sql / .down.sql, where NNNNNN is a
// six digit sequence starting at 000001 without gaps. The baseline is fixed as 000001_baseline.
package migrations

import "embed"

// VersionTable is the table where golang-migrate records the applied version of this service.
const VersionTable = "queue_schema_migrations"

// FS holds the embedded migration files.
//
//go:embed *.sql
var FS embed.FS
