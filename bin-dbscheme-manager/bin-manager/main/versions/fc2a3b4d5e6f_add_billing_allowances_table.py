"""add_billing_allowances_table

Revision ID: fc2a3b4d5e6f
Revises: fd9ebdbd7baa
Create Date: 2026-02-14 12:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'fc2a3b4d5e6f'
down_revision = 'fd9ebdbd7baa'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        CREATE TABLE IF NOT EXISTS billing_allowances (
            id              BINARY(16) PRIMARY KEY,
            customer_id     BINARY(16) NOT NULL,
            account_id      BINARY(16) NOT NULL,
            cycle_start     DATETIME(6) NOT NULL,
            cycle_end       DATETIME(6) NOT NULL,
            tokens_total    INT NOT NULL,
            tokens_used     INT NOT NULL DEFAULT 0,
            tm_create       DATETIME(6) NOT NULL,
            tm_update       DATETIME(6) NOT NULL,
            tm_delete       DATETIME(6) NOT NULL DEFAULT '9999-01-01 00:00:00.000000',
            INDEX idx_billing_allowances_customer_id (customer_id),
            INDEX idx_billing_allowances_account_id (account_id),
            UNIQUE INDEX idx_billing_allowances_account_cycle (account_id, cycle_start)
        );
    """)


def downgrade():
    op.execute("""DROP TABLE IF EXISTS billing_allowances;""")
