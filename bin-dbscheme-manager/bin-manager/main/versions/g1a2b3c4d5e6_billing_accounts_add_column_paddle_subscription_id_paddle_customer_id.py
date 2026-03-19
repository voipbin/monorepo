"""billing_accounts_add_column_paddle_subscription_id_paddle_customer_id

Revision ID: g1a2b3c4d5e6
Revises: ffb1bfe3f8d7
Create Date: 2026-03-19 12:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'g1a2b3c4d5e6'
down_revision = 'ffb1bfe3f8d7'
branch_labels = None
depends_on = None


def _column_exists(conn, table, column):
    result = conn.execute(sa.text(
        "SELECT COUNT(*) FROM information_schema.columns "
        "WHERE table_schema = DATABASE() AND table_name = :table AND column_name = :col"
    ), {'table': table, 'col': column})
    return result.scalar() > 0


def _index_exists(conn, table, index):
    result = conn.execute(sa.text(
        "SELECT COUNT(*) FROM information_schema.statistics "
        "WHERE table_schema = DATABASE() AND table_name = :table AND index_name = :idx"
    ), {'table': table, 'idx': index})
    return result.scalar() > 0


def upgrade():
    conn = op.get_bind()

    if not _column_exists(conn, 'billing_accounts', 'paddle_subscription_id'):
        op.execute("""
            ALTER TABLE billing_accounts
            ADD COLUMN paddle_subscription_id VARCHAR(255) DEFAULT NULL
        """)

    if not _column_exists(conn, 'billing_accounts', 'paddle_customer_id'):
        op.execute("""
            ALTER TABLE billing_accounts
            ADD COLUMN paddle_customer_id VARCHAR(255) DEFAULT NULL
        """)

    if not _index_exists(conn, 'billing_accounts', 'ix_billing_accounts_paddle_subscription_id'):
        op.execute("""
            CREATE INDEX ix_billing_accounts_paddle_subscription_id
            ON billing_accounts(paddle_subscription_id)
        """)

    if not _index_exists(conn, 'billing_accounts', 'ix_billing_accounts_paddle_customer_id'):
        op.execute("""
            CREATE INDEX ix_billing_accounts_paddle_customer_id
            ON billing_accounts(paddle_customer_id)
        """)

    if not _index_exists(conn, 'billing_billings', 'ix_billing_billings_idempotency_key'):
        op.execute("""
            CREATE INDEX ix_billing_billings_idempotency_key
            ON billing_billings(idempotency_key(255))
        """)


def downgrade():
    conn = op.get_bind()

    if _index_exists(conn, 'billing_billings', 'ix_billing_billings_idempotency_key'):
        op.execute("""
            DROP INDEX ix_billing_billings_idempotency_key ON billing_billings
        """)

    if _index_exists(conn, 'billing_accounts', 'ix_billing_accounts_paddle_customer_id'):
        op.execute("""
            DROP INDEX ix_billing_accounts_paddle_customer_id ON billing_accounts
        """)

    if _index_exists(conn, 'billing_accounts', 'ix_billing_accounts_paddle_subscription_id'):
        op.execute("""
            DROP INDEX ix_billing_accounts_paddle_subscription_id ON billing_accounts
        """)

    if _column_exists(conn, 'billing_accounts', 'paddle_customer_id'):
        op.execute("""
            ALTER TABLE billing_accounts
            DROP COLUMN paddle_customer_id
        """)

    if _column_exists(conn, 'billing_accounts', 'paddle_subscription_id'):
        op.execute("""
            ALTER TABLE billing_accounts
            DROP COLUMN paddle_subscription_id
        """)
