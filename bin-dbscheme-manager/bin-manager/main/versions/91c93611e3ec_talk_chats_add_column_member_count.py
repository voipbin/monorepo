"""talk_chats add column member_count

Revision ID: 91c93611e3ec
Revises: f6d7d64e259e
Create Date: 2026-01-20 04:22:05.458291

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '91c93611e3ec'
down_revision = 'f6d7d64e259e'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE talk_chats
        ADD COLUMN member_count INT NOT NULL DEFAULT 0 AFTER detail
    """)


def downgrade():
    op.execute("""
        ALTER TABLE talk_chats
        DROP COLUMN member_count
    """)
