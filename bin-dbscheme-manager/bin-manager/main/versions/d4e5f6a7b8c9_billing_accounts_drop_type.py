"""billing_accounts_drop_type

Revision ID: d4e5f6a7b8c9
Revises: c3a7f1b2d8e4
Create Date: 2026-02-10 18:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'd4e5f6a7b8c9'
down_revision = 'c3a7f1b2d8e4'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE billing_accounts
            DROP COLUMN type;
    """)


def downgrade():
    op.execute("""
        ALTER TABLE billing_accounts
            ADD COLUMN type VARCHAR(255) NOT NULL DEFAULT 'normal' AFTER detail;
    """)
