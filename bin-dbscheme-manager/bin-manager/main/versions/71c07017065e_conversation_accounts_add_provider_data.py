"""conversation_accounts_add_provider_data

Revision ID: 71c07017065e
Revises: a3c811eb1d4c
Create Date: 2026-05-20 21:23:05.532842

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '71c07017065e'
down_revision = 'a3c811eb1d4c'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE conversation_accounts
            ADD COLUMN provider_data JSON
            COMMENT 'Provider-specific credentials (JSON). WhatsApp: phone_number_id, app_secret.'
    """)


def downgrade():
    op.execute("""
        ALTER TABLE conversation_accounts
            DROP COLUMN provider_data
    """)
