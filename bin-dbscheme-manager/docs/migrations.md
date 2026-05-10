# Migrations

This document covers how to create, name, and verify Alembic migration files for the `voipbin` and `asterisk` databases.

## Creating a Migration

Always use `alembic revision` to generate migration files. Never create them by hand — Alembic generates a collision-free revision ID automatically and correctly chains `down_revision` to the current head.

**Step-by-step:**

```bash
# For the voipbin database (most service tables)
cd bin-dbscheme-manager/bin-manager
alembic -c alembic.ini revision -m "customers_add_column_email"
# → creates main/versions/<id>_customers_add_column_email.py

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
