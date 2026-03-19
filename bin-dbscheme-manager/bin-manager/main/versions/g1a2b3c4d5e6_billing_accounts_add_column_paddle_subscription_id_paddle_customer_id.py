"""billing_accounts_add_column_paddle_subscription_id_paddle_customer_id

Revision ID: g1a2b3c4d5e6
Revises: ffb1bfe3f8d7
Create Date: 2026-03-19 12:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'g1a2b3c4d5e6'
down_revision = 'ffb1bfe3f8d7'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE billing_accounts
        ADD COLUMN paddle_subscription_id VARCHAR(255) DEFAULT NULL,
        ADD COLUMN paddle_customer_id VARCHAR(255) DEFAULT NULL;
    """)

    op.execute("""
        CREATE INDEX ix_billing_accounts_paddle_subscription_id
        ON billing_accounts(paddle_subscription_id);
    """)

    op.execute("""
        CREATE INDEX ix_billing_accounts_paddle_customer_id
        ON billing_accounts(paddle_customer_id);
    """)

    op.execute("""
        CREATE INDEX ix_billing_billings_idempotency_key
        ON billing_billings(idempotency_key);
    """)


def downgrade():
    op.execute("""
        DROP INDEX ix_billing_billings_idempotency_key ON billing_billings;
    """)

    op.execute("""
        DROP INDEX ix_billing_accounts_paddle_customer_id ON billing_accounts;
    """)

    op.execute("""
        DROP INDEX ix_billing_accounts_paddle_subscription_id ON billing_accounts;
    """)

    op.execute("""
        ALTER TABLE billing_accounts
        DROP COLUMN paddle_customer_id,
        DROP COLUMN paddle_subscription_id;
    """)
