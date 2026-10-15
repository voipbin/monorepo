// Package mysqlmigrations embeds the MySQL schema migration files of bin-timeline-manager (the
// tables of the analysis store). The ClickHouse migrations of the event store stay in the parent
// directory and are not part of this package.
//
// File format: NNNNNN_<table>_<verb>_<type>_<items>.up.sql / .down.sql, where NNNNNN is a
// six digit sequence starting at 000001 without gaps. The baseline is fixed as 000001_baseline.
package mysqlmigrations

import "embed"

// VersionTable is the table where golang-migrate records the applied version of this service.
const VersionTable = "timeline_schema_migrations"

// FS holds the embedded migration files.
//
//go:embed *.sql
var FS embed.FS
