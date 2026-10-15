# migrationcheck: utf8mb4_0900_ai_ci table option and verify allowed differences (VOIP-1601 P0c)

## 1. Ticket review

Partly valid. Verified on origin/main: `migrationcheck/check.go` has no constant for `DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci`. `verify-service-migrations.sh` compares the Alembic head database (E1, with the service migrations applied) and the empty database (E2) with a strict `diff`; `ALLOWED_DIFF` works only together with `REFERENCE_DUMP`. A named foreign key (`CONSTRAINT <name> FOREIGN KEY`) is not in `forbiddenWords` and `CONSTRAINT` is a non column keyword, so it is already accepted (test only).

## 2. Problem

A baseline that must give its tables the collation of an existing database (`utf8mb4_0900_ai_ci`, where the Alembic head, which writes only `DEFAULT CHARSET=utf8mb4`, gives the engine default) cannot be written, and the E1 versus E2 check cannot accept an intended difference.

Measured on a scratch MariaDB 12.3 (database `utf8_general_ci`): `DEFAULT CHARSET=utf8mb4` gives `utf8mb4_uca1400_ai_ci`; `DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci` is accepted and gives `utf8mb4_0900_ai_ci`, both on `TABLE_COLLATION` and on every string column. So the E1 versus E2 diff of such a table is the `TBL` line plus the `COL` line of every string column (the tables hold no table name in `COL` lines), not one line. On MySQL 8.0 the engine default is already 0900: measured in the end to end run, E1 and E2 are equal there and the accepted lines stay unused.

## 3. Change (additions only, existing behavior unchanged)

- `migrationcheck`: new constant `OptionUTF8MB4Collate0900AI = "DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci"` added to the constant block and to the `allowed` map in `checkBaselineTables` (the fixed list). The list stays fixed; a service only declares which table uses it.
- `verify-service-migrations.sh`: new variable `E1_ALLOWED_DIFF` (details below).
  - When set, the comparison of `e1-after` (E1 with the service migrations applied) and `e2` (the V1/V3 line) uses `compare_dumps` inside the existing `if`: an accepted line is printed as `ALLOWED-DIFF`, an unused one as `UNUSED-ALLOWED-DIFF` (not a failure, like `ALLOWED_DIFF`; an engine without the difference leaves them unused), any other difference fails.
  - When not set the strict `diff` stays.
  - A separate variable (not `ALLOWED_DIFF`) is used on purpose: `ALLOWED_DIFF` belongs to `REFERENCE_DUMP`, whose current behavior stays unchanged (this change does not touch the reference comparison); the header and the docs state the two uses side by side.
  - A missing or non regular file, or a file without a single `< ` or `> ` line, stops the run before any container starts (a typo cannot silently turn the list into strict mode).
  - As with `ALLOWED_DIFF`, accepted lines apply to every table of the dump (the `COL` and `DEF` lines carry no table name), so the service lists only the lines its intended difference needs, and the PR body names the tables each accepted line affects.
  - `compare_dumps` pairs lines with `comm` after `sort` (no `uniq`), so a line that appears in the diff N times (the same `TBL` or `COL` line of N tables) must be listed N times; the docs and the header say so, and a `--self-test` case with a duplicated line pins it.
  - Not changed: `compare_dumps` itself, so the `ALLOWED_DIFF` and `REFERENCE_DUMP` meaning stays.
  - The `ALLOWED_DIFF is set without REFERENCE_DUMP` message also names `E1_ALLOWED_DIFF`.
  - Pair check in `check_allowed_file` (tab separated, pattern `^[<>] (TBL|COL)\t`, a message that names the tab requirement): the `<` and `>` lines must have the same count, and after blanking the collation field (`COL`: the 8th field, `TBL`: the 3rd) the two multisets must be equal, so only a collation difference can be accepted and a type, nullability or comment change is refused; a `--self-test` negative case with a differing type pins it.
  - The success line of the comparison says how many accepted lines were used instead of "diff 0".
  - Limit: an extra accepted line that nothing needs (stale; on an engine without the difference) is reported as `UNUSED-ALLOWED-DIFF`, not a failure (a needed line that is wrong fails on the engine that has the difference); the PR body says that `E1_ALLOWED_DIFF` is a deliberate separate variable (the card named `ALLOWED_DIFF`), carries no DB names or dump values of an existing database, and attaches the per engine `ALLOWED-DIFF` and `UNUSED-ALLOWED-DIFF` output, and checks the number of listed repeats against the number of tables that declare the 0900 option.
  - One file serves both engines; if a measured engine needs different lines the service says so in its PR.
  - The V2 check (`e1-before` versus `e1-after`, baseline only service) stays strict: an `IF NOT EXISTS` baseline does not change the existing tables.
- The comparison against `REFERENCE_DUMP` is unchanged: diff 0 apart from `ALLOWED_DIFF`.
- Tests: positive and negative cases in `check_test.go` (new option accepted in the shape of the existing option cases, wrong tail and a near miss string such as `utf8mb4_0900_as_ci` rejected, undeclared table rejected; the comparison is case insensitive like for the other options, which stays). Named foreign keys are already covered by the existing `named foreign key` case; only the missing variants (backtick quoted name, `ON DELETE CASCADE`) are added; a `--self-test` case that the accepted lines of an E1 style pair pass and print. The case sits next to the other `compare_dumps` cases, before the container loop, so it needs no docker. The file check is a function defined before the `--self-test` branch, next to `compare_dumps`, `check_allowed_file` (regular file, at least one `< ` or `> ` line, and every such line starts with `TBL` or `COL` after the marker: the collation differences of this change appear only there, so a line of the `CON`, `KCU`, `FK`, `IDX`, `CHK`, `DEF` or `GEN` tags, which guard names, keys and rules, is refused) used by the startup validation and by `--self-test` negative cases (missing file, comments only, a `FK` or `CON` line), so it needs no docker; the wiring of the comparison is covered by the end to end runs in section 6. Only those cases run without docker; `--self-test` as a whole still starts engine containers. Not needed: `HELPER_TESTS=1` (no `migrationhandler` DB behavior changes).
- Docs: `docs/workflows/service-owned-migrations.md` (rule 2: the sentence "only what Alembic declared is carried over" is reworded (the 0900 string reproduces the collation of an existing database that the Alembic text left to the engine default), five strings; rule 5: the checker accepts a named foreign key and the `CON`, `KCU`, `FK` dump lines are the final guard of the names; the Checks list and the "accepted only when identical" sentence: E1_ALLOWED_DIFF next to ALLOWED_DIFF, with the TBL/COL and collation only limits), the header of the script, `bin-common-handler/docs/architecture.md` (line 96 stays: it already says "table option allow list").

## 4. Out of scope

Service baselines, `.circleci`, the existing constants and checks, relaxing the reference dump comparison.

## 5. Impact analysis (no schema or runtime change)

Static text check in unit tests and a developer script; no DDL runs in production code, no startup behavior changes. Old and new service code are unaffected: no existing `Options` value changes meaning and the script default is unchanged.

## 6. Verification

The check accepts any pair that differs only in the collation field, not only the 0900 one.

`go test` of migrationcheck (positive and negative), the 5 step workflow in bin-common-handler, build of all dependent modules, bin-tag-manager `go test ./migrations/` and `verify-service-migrations.sh bin-tag-manager` (both engines). End to end on a scratch copy: a baseline with the 0900 option fails without `E1_ALLOWED_DIFF` and passes with it, printing the accepted lines, on MariaDB 12.3 and MySQL 8.0.
