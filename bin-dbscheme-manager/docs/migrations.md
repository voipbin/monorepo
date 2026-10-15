# Migrations

> **Freeze notice:** The `bin-manager` stream is frozen. Do not add new Alembic migrations for the `bin_manager` database; every application table there is changed only through the owning service's `migrations/` directory (see [schema-ownership.md](schema-ownership.md)). The `asterisk_config` stream is not frozen and keeps using Alembic.

This document covers how to create, name, and verify Alembic migration files for the `bin_manager` and `asterisk` databases.

## Creating a Migration

Always use `alembic revision` to generate migration files. Never create them by hand — Alembic generates a collision-free revision ID automatically and correctly chains `down_revision` to the current head.

**Step-by-step:**

```bash
# The bin_manager stream (bin-dbscheme-manager/bin-manager) is frozen: no new revisions.

# For the asterisk database (PJSIP config tables)
cd bin-dbscheme-manager/asterisk_config
alembic -c alembic.ini revision -m "ps_endpoints_add_column_dtls_auto_gen"
# → creates config/versions/<id>_ps_endpoints_add_column_dtls_auto_gen.py
```

After generating the file, open it and fill in the `upgrade()` and `downgrade()` functions with raw SQL via `op.execute()`.

**Check there is exactly one head before pushing:**

```bash
alembic -c alembic.ini heads
# Should print exactly one revision hash
```

**Important restrictions:**

- NEVER hand-create migration files with made-up revision IDs — they collide with existing migrations.
- NEVER run `alembic upgrade` against staging or production without human authorization and VPN access.
- NEVER run `alembic downgrade` in production without human authorization.

## Naming Conventions

Use snake_case. Follow this pattern:

```
<table_name>_<verb>_<type>_<items>
```

**Verbs:** `create`, `add`, `remove`, `alter`, `drop`

**Types:** `table`, `column`, `index`

**Examples:**

```
customers_add_column_email
registrar_trunks_create_table
ai_aicalls_remove_column_engine_type
call_outbound_configs_add_index_customer_id
```

When a migration touches multiple tables, list the primary table first:

```
billing_accounts_add_billing_allowances_table
```

## Example Migration

A complete migration that creates a new table with proper `upgrade()` and `downgrade()` functions:

```python
"""recordings_create_table

Revision ID: a3f1c9e87d42
Revises: 044ac45bc2e3
Create Date: 2024-03-15 10:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'a3f1c9e87d42'
down_revision = '044ac45bc2e3'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        CREATE TABLE call_recordings (
            id          BINARY(16)   NOT NULL,
            customer_id BINARY(16)   NOT NULL,
            call_id     BINARY(16)   NOT NULL,

            format      VARCHAR(32)  NOT NULL DEFAULT '',
            duration    INT          NOT NULL DEFAULT 0,
            uri         VARCHAR(512) NOT NULL DEFAULT '',

            tm_create   DATETIME(6),
            tm_update   DATETIME(6),
            tm_delete   DATETIME(6) DEFAULT '9999-01-01 00:00:00.000000',

            PRIMARY KEY (id)
        );
    """)
    op.execute("""
        CREATE INDEX idx_call_recordings_customer_id ON call_recordings(customer_id);
    """)
    op.execute("""
        CREATE INDEX idx_call_recordings_call_id ON call_recordings(call_id);
    """)


def downgrade():
    op.execute("""DROP TABLE IF EXISTS call_recordings;""")
```

**Key rules visible in this example:**

- All UUID columns (`id`, `customer_id`, `call_id`) use `BINARY(16)`, never `VARCHAR(36)`.
- `downgrade()` always mirrors `upgrade()` — drop what was created, remove what was added.
- Raw SQL via `op.execute()` is the project convention (not SQLAlchemy DDL helpers).
- Soft-delete sentinel: `tm_delete DEFAULT '9999-01-01 00:00:00.000000'` is the standard pattern.

## CI: `migration-lint` (automatic, every branch)

Every branch/PR that touches `bin-dbscheme-manager/**` runs a CircleCI job
called `migration-lint`, with no approval gate. It does **not** require the
main-only `migration-applied-checkpoint` approval and runs independently of
it.

**What it checks**, for both migration streams (`bin-manager/` for the
`bin_manager` database, `asterisk_config/` for the `asterisk` database):

1. Python syntax/import validity of every migration file in `versions/`.
2. Exactly one head per stream (`alembic heads`) — more than one head means
   two migrations were authored against the same `down_revision` without one
   rebasing onto the other (see "Check there is exactly one head before
   pushing" above).
3. No unresolvable `down_revision` references (`alembic history` walks the
   full chain from base to head).

This is a **metadata-only** check — `alembic heads`/`alembic history` never
open a database connection, so `migration-lint` needs no DB, no Docker, no
secrets. It cannot and does not apply any migration to any database; that
remains a manual, VPN-only, human-only procedure gated by
`migration-applied-checkpoint` (main-only, see `docs/operations.md`).

**Where to look when `migration-lint` fails**: the CircleCI job's own log
output names the specific problem — a multiple-heads error (pointing back to
this document), a Python import/syntax traceback from `alembic heads`, or an
`alembic.util.exc.CommandError` citing the broken `down_revision` reference.
No VPN access or database credential is needed to diagnose any of these; fix
the migration file(s) locally and push again.

## Writing migrations that can be applied automatically

A revision is applied without a person only when it is provably additive. Anything else is not rejected by CI; it is classified as needing a person to apply it. To stay in the automatic path:

- `upgrade()` calls `op.execute()` with raw SQL, one statement per call, built only from constants (string literals, `+`, `%s`/`%d` formatting, f-strings of constants, `for` and `if` over constants).
- Allowed statements:
  - `CREATE TABLE [IF NOT EXISTS]` with column definitions, `PRIMARY KEY`, `KEY`/`INDEX`, `UNIQUE KEY`, `CONSTRAINT <name> UNIQUE|PRIMARY KEY`, and the table options `ENGINE=InnoDB`, `ROW_FORMAT`, `CHARSET`, `COLLATE`, `COMMENT`.
  - `CREATE INDEX <name> ON <table> (<columns>)`.
  - `ALTER TABLE <table> ADD [COLUMN] [IF NOT EXISTS] <name> <type> <attributes>` and `ALTER TABLE <table> ADD INDEX|KEY <name> (<columns>)`.
- A new `NOT NULL` column needs a `DEFAULT` other than `NULL`. Attributes are `NULL`/`NOT NULL`, `DEFAULT`, `COMMENT`, `UNSIGNED`, `CHARACTER SET`, `COLLATE` and `AFTER <column>`, each at most once. Defaults are `NULL`, numbers, quoted strings, `true`/`false`, `0x...`, `CURRENT_TIMESTAMP[(n)]`, `now()`, `(JSON_OBJECT())`, `(JSON_ARRAY())` and `(UNHEX('<hex>'))`.
- Write a unique index on a new table as `UNIQUE KEY` inside `CREATE TABLE`; a separate `CREATE UNIQUE INDEX` needs a person.
- A SQL comment is `-- ` followed by a space, and it must end with a line feed inside the statement. Do not put a comment at the very end of a statement, and do not use `#` or `/* */`.
- Do not write a colon followed by a word (`:name`) outside a word, for example a bind parameter. A colon after a word character, as in `'12:30:00'` or `'ratio 1:2'`, is fine.
- Keep the generated file shape: the template imports, a docstring, plain assignments of `revision`, `down_revision` (a single revision) and `depends_on = None`, `branch_labels = None`, then `upgrade()` and `downgrade()`. `downgrade()` is never run by the classifier.

The source gate rejects these constructs (the revision then needs a person): `get_bind`, `get_context`, `inspect`/`Inspector`, `batch_alter_table`, `op.alter_column`, `op.drop_*`, `op.create_foreign_key`, `op.create_unique_constraint`, `os`/`getenv`, `getattr`/`setattr`, `eval`/`exec`/`compile`/`__import__`, raw DBAPI cursors, sessions, `print`, `sa.func`, `sa.true`/`sa.false`, `sa.PrimaryKeyConstraint`, `sa.UniqueConstraint`, `sa.ForeignKey`, `try`, `with`, `lambda`, comprehensions, nested functions, helper functions, `class`, `global`, `branch_labels`, a tuple `down_revision`, `depends_on`, decorators, and any call or statement at module level. Source files must be UTF-8 without a byte order mark or a coding declaration, with ASCII identifiers; non-ASCII text is allowed inside comments, docstrings and string literals when it is visible text.

Revision files that predate these rules are listed by sha256 in `classifier/module_level_exceptions.txt`. The list exempts an already applied file from the import-time rules only; a changed file is not covered, and a listed file that becomes pending is still treated as needing a person. A new revision must satisfy the rules instead of being added to the list.

## Classifier tests

`pytest bin-dbscheme-manager/tests` covers the tokenizer, `normalize()`, the statement allow-list, the source gate (accepted and rejected forms), the per-stream classification, the runtime replay guard, and the real revision files. The real-revision tests apply both streams to a scratch MariaDB 12.3 server and check that the offline render equals what is executed, and that every statement classified additive only adds to the schema. See `CLAUDE.md` for how to run them. CI runs them in the `bin-dbscheme-manager-test` job on every branch.
