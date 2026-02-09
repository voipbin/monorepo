"""billing_accounts_add_plan_type

Revision ID: c3a7f1b2d8e4
Revises: fb5b8d1359c9
Create Date: 2026-02-09 18:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'c3a7f1b2d8e4'
down_revision = 'fb5b8d1359c9'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE billing_accounts
            ADD COLUMN plan_type VARCHAR(255) NOT NULL DEFAULT 'free' AFTER type;
    """)


def downgrade():
    op.execute("""
        ALTER TABLE billing_accounts
            DROP COLUMN plan_type;
    """)
