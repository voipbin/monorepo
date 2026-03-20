"""billing_billings_idempotency_key_unique

Convert billing_billings.idempotency_key from TINYBLOB to VARBINARY(16) and replace
the non-unique index with a UNIQUE index to enforce idempotency at the database level.
This prevents duplicate billing records from concurrent Paddle webhook deliveries
(TOCTOU race in the application-level check).

The column was originally created as sa.LargeBinary(16) which maps to TINYBLOB in MySQL.
BLOB/TEXT columns cannot have a UNIQUE index without a prefix length, and prefix-length
UNIQUE indexes do not enforce full-value uniqueness. VARBINARY(16) stores the same binary
UUID data but supports full UNIQUE indexes.

Pre-Paddle billing records store zero-value UUIDs (16 zero bytes) rather than NULL.
These are converted to NULL before creating the index, since MySQL allows multiple
NULL values in a UNIQUE index but not multiple identical non-NULL values.

Revision ID: h2b3c4d5e6f7
Revises: g1a2b3c4d5e6
Create Date: 2026-03-20 12:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'h2b3c4d5e6f7'
down_revision = 'g1a2b3c4d5e6'
branch_labels = None
depends_on = None


def _index_exists(conn, table, index):
    result = conn.execute(sa.text(
        "SELECT COUNT(*) FROM information_schema.statistics "
        "WHERE table_schema = DATABASE() AND table_name = :table AND index_name = :idx"
    ), {'table': table, 'idx': index})
    return result.scalar() > 0


def upgrade():
    conn = op.get_bind()

    # Drop the old non-unique index (must drop before altering column type)
    if _index_exists(conn, 'billing_billings', 'ix_billing_billings_idempotency_key'):
        op.execute("""
            DROP INDEX ix_billing_billings_idempotency_key ON billing_billings
        """)

    # Convert TINYBLOB → VARBINARY(16) so MySQL can create a full UNIQUE index.
    # BLOB/TEXT columns only support prefix indexes which cannot enforce uniqueness.
    op.execute("""
        ALTER TABLE billing_billings
        MODIFY COLUMN idempotency_key VARBINARY(16) NULL
    """)

    # Pre-Paddle billing records have zero-value UUIDs (Go's uuid.UUID zero value:
    # 16 zero bytes) instead of NULL. Convert them to NULL so the UNIQUE index
    # allows multiple old records (MySQL permits multiple NULLs in UNIQUE indexes).
    op.execute("""
        UPDATE billing_billings
        SET idempotency_key = NULL
        WHERE idempotency_key = X'00000000000000000000000000000000'
    """)

    # Create UNIQUE index. MySQL allows multiple NULLs in a UNIQUE index,
    # so existing rows with NULL idempotency_key are unaffected.
    if not _index_exists(conn, 'billing_billings', 'ux_billing_billings_idempotency_key'):
        op.execute("""
            CREATE UNIQUE INDEX ux_billing_billings_idempotency_key
            ON billing_billings(idempotency_key)
        """)


def downgrade():
    conn = op.get_bind()

    # Revert to non-unique index
    if _index_exists(conn, 'billing_billings', 'ux_billing_billings_idempotency_key'):
        op.execute("""
            DROP INDEX ux_billing_billings_idempotency_key ON billing_billings
        """)

    # Revert VARBINARY(16) → TINYBLOB
    op.execute("""
        ALTER TABLE billing_billings
        MODIFY COLUMN idempotency_key TINYBLOB NULL
    """)

    if not _index_exists(conn, 'billing_billings', 'ix_billing_billings_idempotency_key'):
        op.execute("""
            CREATE INDEX ix_billing_billings_idempotency_key
            ON billing_billings(idempotency_key(255))
        """)
