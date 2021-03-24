"""add calls index userid tmcreate

Revision ID: ea88aa5daa8c
Revises: c63a7f966a05
Create Date: 2021-03-25 03:50:21.062325

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ea88aa5daa8c'
down_revision = 'c63a7f966a05'
branch_labels = None
depends_on = None


def upgrade():
    op.create_index('idx_calls_userid_tmcreate', 'calls', ['user_id', 'tm_create'])

def downgrade():
    op.drop_index('idx_calls_userid_tmcreate', table_name='calls')
