"""ai_messages_add_column_active_ai_id

Revision ID: ef130bcf8401
Revises: 71c07017065e
Create Date: 2026-05-21 19:51:17.498429

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ef130bcf8401'
down_revision = '71c07017065e'
branch_labels = None
depends_on = None


def upgrade():
    op.add_column('ai_messages',
        sa.Column('active_ai_id', sa.BINARY(16), nullable=False,
                  server_default=sa.text("0x00000000000000000000000000000000")))


def downgrade():
    op.drop_column('ai_messages', 'active_ai_id')
