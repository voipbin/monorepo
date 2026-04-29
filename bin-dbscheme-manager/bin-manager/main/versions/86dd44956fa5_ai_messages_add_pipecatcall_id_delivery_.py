"""ai_messages add pipecatcall_id delivery_status

Revision ID: 86dd44956fa5
Revises: 629a307b92d0
Create Date: 2026-04-29 12:29:35.225461

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import mysql


# revision identifiers, used by Alembic.
revision = '86dd44956fa5'
down_revision = '629a307b92d0'
branch_labels = None
depends_on = None


def upgrade():
    op.add_column('ai_messages',
        sa.Column('pipecatcall_id', mysql.BINARY(16), nullable=True))
    op.add_column('ai_messages',
        sa.Column('delivery_status', sa.String(16), nullable=False,
                  server_default='delivered'))
    op.create_index('idx_ai_messages_pcc_delivery', 'ai_messages',
                    ['pipecatcall_id', 'delivery_status'])


def downgrade():
    op.drop_index('idx_ai_messages_pcc_delivery', 'ai_messages')
    op.drop_column('ai_messages', 'delivery_status')
    op.drop_column('ai_messages', 'pipecatcall_id')
