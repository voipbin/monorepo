"""billing_billings_idempotency_key_unique

Replace the non-unique index on billing_billings.idempotency_key with a UNIQUE index
to enforce idempotency at the database level. This prevents duplicate billing records
from concurrent Paddle webhook deliveries (TOCTOU race in the application-level check).

MySQL allows multiple NULL values in a UNIQUE index, so existing rows with
NULL idempotency_key (pre-Paddle billing records) are unaffected.

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

    # Drop the old non-unique index
    if _index_exists(conn, 'billing_billings', 'ix_billing_billings_idempotency_key'):
        op.execute("""
            DROP INDEX ix_billing_billings_idempotency_key ON billing_billings
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

    if not _index_exists(conn, 'billing_billings', 'ix_billing_billings_idempotency_key'):
        op.execute("""
            CREATE INDEX ix_billing_billings_idempotency_key
            ON billing_billings(idempotency_key(255))
        """)
