"""call_outbound_configs_uuid_columns_to_binary

Revision ID: dd500d4e8cb7
Revises: 98abdec1a745
Create Date: 2026-05-07 23:04:41.422972

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'dd500d4e8cb7'
down_revision = '98abdec1a745'
branch_labels = None
depends_on = None


def upgrade():
    # Bug: call_outbound_configs.id and customer_id were created as VARCHAR(36),
    # but the rest of the monorepo (customer_customers.id, billing_accounts.id,
    # etc.) uses BINARY(16) for UUIDs. The Go MySQL driver sends uuid.UUID as
    # 16 raw bytes — those bytes were silently stored as garbage characters
    # in the VARCHAR(36) columns, and joins/lookups across services failed.
    #
    # Fix: drop existing (corrupted) rows, change the columns to BINARY(16),
    # then re-bootstrap an empty config for every active customer.

    # Wipe rows — they cannot be reliably converted because the byte
    # encoding from the bootstrap migration vs API inserts may differ.
    op.execute("DELETE FROM call_outbound_configs")

    # Drop unique key so we can ALTER the underlying column type.
    op.execute("ALTER TABLE call_outbound_configs DROP INDEX uq_customer_id")

    # Convert UUID columns to BINARY(16), matching monorepo convention.
    op.execute("ALTER TABLE call_outbound_configs MODIFY id BINARY(16) NOT NULL")
    op.execute("ALTER TABLE call_outbound_configs MODIFY customer_id BINARY(16) NOT NULL")

    # Restore unique key on the new column type.
    op.execute("ALTER TABLE call_outbound_configs ADD UNIQUE KEY uq_customer_id (customer_id)")

    # Re-bootstrap an empty config for every active customer.
    # UNHEX(REPLACE(UUID(),'-','')) produces a 16-byte UUID for `id`;
    # `c.id` is already BINARY(16) on customer_customers so it slots in directly.
    # INSERT IGNORE makes this idempotent against a partial earlier bootstrap.
    op.execute("""
        INSERT IGNORE INTO call_outbound_configs
            (id, customer_id, name, detail, destination_whitelist, codecs, tm_create, tm_update)
        SELECT UNHEX(REPLACE(UUID(), '-', '')), c.id, '', '', '[]', '', NOW(), NOW()
        FROM customer_customers c
        WHERE c.tm_delete IS NULL
    """)


def downgrade():
    # Wipe rows (they would not survive a column-type change either way).
    op.execute("DELETE FROM call_outbound_configs")

    op.execute("ALTER TABLE call_outbound_configs DROP INDEX uq_customer_id")
    op.execute("ALTER TABLE call_outbound_configs MODIFY id VARCHAR(36) NOT NULL")
    op.execute("ALTER TABLE call_outbound_configs MODIFY customer_id VARCHAR(36) NOT NULL")
    op.execute("ALTER TABLE call_outbound_configs ADD UNIQUE KEY uq_customer_id (customer_id)")
