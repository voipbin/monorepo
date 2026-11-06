# VOIP-1686: email_configs migration 000002 fails on a database with the uca1400 default collation

## 1. Ticket review (analysis)

Verdict: valid. Checked against main and scratch containers (MariaDB 12.3.3, MySQL 8.0.46).

- Reproduced: on a database created with `CREATE DATABASE x CHARACTER SET utf8` (MariaDB 12.3 resolves it to `utf8mb3_uca1400_ai_ci`), `000002_email_configs_create_table_all.up.sql` fails with `Error 1267 (HY000): Illegal mix of collations (utf8mb3_general_ci,COERCIBLE) and (utf8mb3_uca1400_ai_ci,COERCIBLE) for operation '<>'`.
- The only failing expression is `JSON_TYPE(value_json) <> 'NULL'`. `JSON_TYPE` returns the system collation `utf8mb3_general_ci`; the literal takes the database default. The other CHECKs compare columns with literals (the column collation wins over a coercible literal) or integers and booleans, so they cannot raise 1267. 000003 has the same kind of CHECKs only.
- A failed `CREATE TABLE` leaves no table on MariaDB, so a failed 000002 leaves no partial schema.
- The same engines with `utf8mb4`, with an explicit `utf8_general_ci`, and MySQL 8.0 with `utf8` all pass, which is why the existing checks (they create the database with `utf8_general_ci`) did not see it.
- Other services: a case-insensitive grep over all `*.sql` finds a function result compared with a string literal only in this file.

Not verified: step V13 applies to every service by script, but it was run (by hand, same steps on scratch containers) only against bin-email-manager (the other services were not run; a generated column using `JSON_UNQUOTE(JSON_EXTRACT(...))` in another baseline is not checked here); the exact engine version and database collation of any deployed environment beyond what the error text shows.

## 2. Design

1. 000002 `chk_email_configs_json` becomes `CHECK (value_json IS NULL OR CAST(JSON_TYPE(value_json) AS BINARY) <> CAST('NULL' AS BINARY))`. A binary comparison has no collation, so it behaves the same on every database default and connection collation, on MariaDB and MySQL. Meaning is unchanged: JSON `null` is rejected; SQL NULL, `{}`, numbers and the JSON string `"NULL"` are accepted.
   Rejected alternatives: `COLLATE utf8mb3_general_ci` (a MariaDB/MySQL specific collation name, fragile when defaults change); `CAST(... AS CHAR)` (takes the connection/database collation again, so it does not remove the dependency).
2. 000003 is unchanged (reasoning above, and applied on the default-collation database).
3. 000002 is edited in place, no new file. A new number cannot help: on a database with the default collation, 000002 would still fail before any later file runs. golang-migrate stores only the version number, no checksum, so an environment where 000002 already succeeded keeps the old constraint text, which is semantically identical; nothing is re-run. This is a one-time exception to "merged migration files are immutable", limited to this one CHECK line. `scripts/check-service-migrations.sh` reports the file as modified (status M); that is the expected result for this PR.
   `migrationcheck.Check` validates file structure only and does not compare content with main (confirmed: `Test_embeddedMigrations` passes after the edit).
4. Guard A (static): `Test_noCollationDependentCompare` in `bin-email-manager/migrations` fails when an up file calls `JSON_TYPE`, `JSON_VALUE`, `JSON_UNQUOTE`, `JSON_EXTRACT`, `LOWER`, `UPPER`, `CONCAT`, `SUBSTRING` or `TRIM` without wrapping the call directly in `CAST(`. It is a narrow, cheap guard for this service only: it ignores comments and the function list is fixed, it does not check what follows the cast (a `CAST(... AS CHAR)` or a one-sided cast passes), and a function wrapped inside another function inside `CAST(` is a false positive. Other functions and other forms are covered by guard B, which every service gets from `verify-service-migrations.sh`. `Test_collationDependentFuncs` covers upper/lower case, a space after `CAST(`, and the cast form.
5. Guard B (dynamic): `scripts/verify-service-migrations.sh` step V13 creates one more database with `CREATE DATABASE e8 CHARACTER SET utf8` only (default collation; this is `utf8mb3_uca1400_ai_ci` on MariaDB 12.x, on engines whose default is `utf8mb3_general_ci` it adds no coverage) and runs up, down to 0, up. It is not compared with the Alembic head (up/down/up only), so collation differences do not affect the E1 comparison. The down/up part is skipped for a service marked DESTRUCTIVE. The database is not dropped; the engine container is discarded at the end of the run. A `utf8mb4` database would not catch the regression, hence `utf8`.
6. `docs/workflows/service-owned-migrations.md` gets one rule line.

## 3. Verify items

- V1 (RED) old 000002 on a `CHARACTER SET utf8` MariaDB 12.3 database fails with 1267.
- V2 After the fix: 000001 to 000003 up, down to 0, up on MariaDB 12.3 (`utf8` default and explicit `utf8_general_ci`) and MySQL 8.0; row checks: JSON null rejected; SQL NULL, `{}`, `1`, `"NULL"` accepted.
- V3 Guard A fails on the old file and passes on the new one.
- V4 Guard B fails with the old 000002 and passes with the fix.
- V5 `go test ./...`, lint, `scripts/check-*.sh`, `scripts/check-service-migrations.sh bin-email-manager` (M expected).

## 4. Impact analysis

- a-1 destructive operations: none. Only the text of a CHECK in a `CREATE TABLE IF NOT EXISTS` changes; no DROP, UPDATE, DELETE, rename or type change.
- a-2 rolling compatibility: the constraint accepts and rejects the same values as before, so old code on the new schema and new code on the old schema behave identically.
- a-3 lock and time: creation of an empty new table, instant.
- a-4 rollback: revert the commit. The down file (unchanged) drops the table and is marked lossy.
- a-5 data loss: none.
