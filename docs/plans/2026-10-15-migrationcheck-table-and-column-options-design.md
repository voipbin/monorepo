# migrationcheck: table option with ROW_FORMAT and column level charset (VOIP-1601 P0b)

## 1. Ticket review

Valid. Alembic table option tails on origin/main: `DEFAULT CHARSET=utf8` 1, `utf8mb4` 9, `utf8mb4 COLLATE=utf8mb4_unicode_ci` 11, the same with `ROW_FORMAT=DYNAMIC` 10. Only the last has no constant in `migrationcheck/check.go`. Alembic `f46d9c5c4438` sets `CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci` on `ai_messages.content`, which the deny list rejects.

## 2. Problem

A baseline cannot reproduce 10 Alembic tables (ROW_FORMAT) or `ai_messages.content`, so the dump comparison (diff 0) cannot pass for those services.

## 3. Change (public API additions only)

- `OptionUTF8MB4UnicodeCIDynamic = "DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC"`, added to the fixed allowed table option list.
- `ColumnCharsetUTF8MB4GeneralCI = "CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci"`, the only allowed column charset string.
- `Options.ColumnCharsets map[string]string`: key `table.column`, value one of the allowed strings.

## 4. Behavior

The definition between the parentheses is split at top level commas on the raw text (quotes, backticks and nested parentheses honored, so `DECIMAL(10,2)`, `ENUM('a,b')`, `KEY k (a,b)` stay in one item). The column name is the first token of an item (backticks removed). An item whose first token is unquoted `PRIMARY`, `KEY`, `UNIQUE`, `INDEX`, `CONSTRAINT`, `FOREIGN`, `FULLTEXT`, `SPATIAL` or `CHECK` is not a column and is only deny list scanned.

- Key format: exactly `table.column`, both parts non empty, else an error. Table and column names compare case sensitively (a mismatch is an error, never a pass).
- Matching works on the `stripQuoted` text of the item (text outside quotes), split with `wordRegexp` into tokens, compared case insensitively. The declared string is its token sequence; the first consecutive occurrence is removed before the deny list scan. A string hidden in a COMMENT therefore never satisfies a declaration, a second occurrence is still rejected by the deny list, and partial words (`utf8mb4_general_ci_x`) do not match.
- A column without a declaration is scanned as before.
- Errors (format `<file>: <table>.<column>: ...`, sorted by key): value not equal to the one allowed constant (exact match, like the table option list); malformed key; declared table not created by the baseline; declared column not found in the table (found over every statement of that table); any definition of the declared column (a table created twice has two) that does not contain the declared string. Nil map or empty value: no declaration, empty value is an error.
- Existing table option check, deny list words and constants are unchanged.

## 5. Out of scope

Other features, service code (including bin-tag-manager), `.circleci`, `scripts`, relaxing the dump comparison. Freeze rule: additions only, no behavior change for existing callers (Options without the new field behave as today).

## 6. Docs

`docs/workflows/service-owned-migrations.md`: rule 1 (line 26, ROW_FORMAT=DYNAMIC is kept through the new option), rule 2 (line 27, four strings and `ColumnCharsets`), the paragraph after rule 7 (column charset is no longer "rejected for now"). `bin-common-handler/docs/architecture.md` line 96 (allow list). bin-tag-manager docs need no change (no use of the new API).

## 7. Verification

Negative tests: string in a COMMENT, second occurrence, partial word, undeclared column charset, other collation, empty value, malformed key, missing table, missing column, declared string absent, `key` backtick column vs `KEY` index, commas in DECIMAL/ENUM/COMMENT/KEY/CHECK/backtick name. Positive tests: new table option, declared column charset, case and white space variants. Then the 5 step verification workflow in bin-common-handler and bin-tag-manager (`go test ./migrations/`), build of dependent modules.
