# Service-owned migrations

A service that owns its tables applies its schema migrations itself at startup (golang-migrate, SQL files embedded with `go:embed`) instead of through Alembic. The shared code lives in `bin-common-handler/pkg/migrationhandler`; `bin-tag-manager` is the reference service. Tested engines: MariaDB 12.3 and MySQL 8.0, so migration files avoid engine specific syntax. The `bin-manager` Alembic stream is frozen (see `bin-dbscheme-manager/CLAUDE.md`); the list of converted services is `bin-dbscheme-manager/docs/schema-ownership.md`.

## What a service holds

- `migrations/embed.go`: `const VersionTable = "<prefix>_schema_migrations"` and `//go:embed *.sql` `var FS embed.FS`
- `migrations/NNNNNN_<description>.up.sql` and `.down.sql` (six digits from 000001 without gaps; the first pair is `000001_baseline`)
- `cmd/<name>-manager/main.go`, right after the database connection and before the cache and the message broker:
  `migrationhandler.Run(cfg.DatabaseDSN, migrations.FS, migrations.VersionTable)`
- `cmd/<name>-control/main.go`: `migrationhandler.NewMigrateCommand("migrate", open)`, where `open` returns `migrationhandler.New(dsn, migrations.FS, migrations.VersionTable)`. The command (`status`, `up`, `down`, `force <version>`) opens only the database, so it works while the cache or the broker are down. A service whose control binary already has a `migrate` command registers it under another name.
- `migrations/migrations_test.go`: `migrationcheck.Check(FS, migrationcheck.Options{TableOptions: ...})` and a test that no non-.sql file sits next to the embedded files. For a migration directory below the service root pass `fs.Sub(FS, "<dir>")`.

## Rules for the files

- One DDL statement per file. Only the baseline may hold several `CREATE TABLE IF NOT EXISTS` statements.
- Every up has a down. A down that cannot restore data starts with `-- DOWN-LOSSY: <reason>`; an up with a destructive operation starts with `-- DESTRUCTIVE: <reason>` and its PR body starts with `[DESTRUCTIVE]`.
- Merged files are immutable and numbers are never reused. Fix a mistake with a new number.
- Expand/contract: the old code must work on the new schema. Drop a column in a later PR than the one that stops using it; replace instead of rename or retype.
- The baseline down is the no-op `SELECT 1`.

## Baseline from the Alembic head

The baseline of a converted service must create what the Alembic head created.

1. `CREATE TABLE` becomes `CREATE TABLE IF NOT EXISTS`; `ENGINE=InnoDB` and `ROW_FORMAT=DYNAMIC` are dropped (database defaults).
2. Table options: only what Alembic declared is carried over, and only as one of the three strings of `migrationcheck` (`OptionUTF8MB4`, `OptionUTF8MB4UnicodeCI`, `OptionUTF8MB3`). The service test lists the table names that use one; a table that is not listed must have no table option. A collation that exists only in MariaDB (`*_uca1400_*`) is not copied.
3. JSON columns are written as `json`; expression defaults are written as `DEFAULT (expr)`. Column comments are kept.
4. Generation expressions are taken from the Alembic source, not from `SHOW CREATE TABLE` (MariaDB and MySQL print them differently).
5. Foreign keys without a name stay without a name (MariaDB names them `1`, `2`, which MySQL rejects).
6. Seed rows are not part of the baseline; a seed is its own file with one statement.
7. A new database must be created with `CHARACTER SET utf8 COLLATE utf8_general_ci`, like the install scripts do, otherwise most tables differ from the Alembic head.

The result is accepted only when the extended normalized dump of the service tables is identical to the one of the Alembic head (`scripts/verify-service-migrations.sh`). `migrationcheck` is a deny list for the baseline text; the dump comparison is the final guard. `migrationcheck` also expects the statement to start with `CREATE TABLE IF NOT EXISTS <name> (` with an unquoted name (strip the backticks and the database prefix of `SHOW CREATE TABLE` output) and rejects the words of its deny list even as unquoted column names (quote such a column with backticks). A column level `CHARACTER SET` or `COLLATE` is rejected by `migrationcheck` for now, and so is any text outside its deny list that changes a collation (for example the `BINARY` column attribute); the dump comparison catches those.

## Checks

- `scripts/check-service-migrations.sh <service-dir> [<migrations-subdir>]` (after `git fetch origin main`): merged `.sql` files are not modified, deleted, renamed or changed in mode; the added files are regular files `NNNNNN_<description>.(up|down).sql` in pairs whose numbers continue origin/main without gaps (from 000001 when origin/main has none there). `--self-test` runs it on synthetic branches. A changed `VersionTable` is not detected.
- `ALEMBIC_VENV=<venv> scripts/verify-service-migrations.sh <service-dir> [<migrations-subdir>]` starts scratch containers of both engines and checks: the dump of the Alembic head plus the service migrations equals the dump of an empty database plus the service migrations; applying them to the Alembic database keeps the row counts (not checked when an up file carries a `-- DESTRUCTIVE:` marker); up, down, up; concurrent start; dirty refusal and force; a database ahead of the image; a restricted account; a password with special characters; the daemon applies the migrations before it connects the cache; the structure test. `HELPER_TESTS=1` also runs the `migrationhandler` integration tests; run it for every change of the helper. Output with SKIP lines or with `ENGINES_OVERRIDE` is not evidence for the default engines (the SKIP line of a reference dump on the engine it was not made with is the one expected exception).
  Variables for services that differ: `CONTROL_BIN`, `SERVICE_BIN`, `VERSION_TABLE`, `MIGRATE_CMD`, `DSN_VAR`, `DAEMON_ENV`, `EXISTING_ROWS_SQL`, `SEED_COMPARE_SQL` (see the header of the script).
- Comparing with an existing database: `REFERENCE_DUMP=<file>` (with `REFERENCE_ENGINE`) compares an externally produced dump with the empty database built by the service migrations, `ALLOWED_DIFF=<file>` lists accepted differences (printed as `ALLOWED-DIFF`); any other difference fails. A reference file is engine specific (MariaDB and MySQL 8.0 print column defaults differently): make it with the engine of the database it describes and name that engine in `REFERENCE_ENGINE` (required); the comparison runs on that engine and is skipped, with a SKIP line, on the other one; the allowed lines apply to every table of the dump, not to one table. The run prints `reference dump not given` when no reference was compared. `scripts/verify-service-migrations.sh --dump <service-dir> <database> [<migrations-subdir>]` produces the dump from a database through `SQL_CMD` (client option file for the credentials).
- `scripts/verify-service-migrations.sh --self-test` exercises the dump queries (generated column, comments, foreign key rules, check expressions) on a synthetic schema.

## Impact analysis for a migration PR

Every PR that changes a migration file or the migration runner states, with evidence, for each point: destructive operations one by one (DROP of any object, TRUNCATE, data changing UPDATE/DELETE/INSERT SELECT, type narrowing or change, default removal or change, rename, NOT NULL/UNIQUE/FK/CHECK on existing data, enum value removal, NOT NULL column without default); compatibility of the old code with the new schema and of the new code with the old schema (grep results); lock and duration with the table size source; rollback (the helper skips a database version that is ahead of the image, so an older image still starts; a rollback of a baseline is the previous image); data loss. A PR with a destructive or ambiguous operation starts its body with `[DESTRUCTIVE]`. Attach the output of the two engine verification.
