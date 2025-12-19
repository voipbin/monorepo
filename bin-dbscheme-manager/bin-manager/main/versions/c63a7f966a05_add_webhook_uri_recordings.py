"""add_webhook_uri_recordings

Revision ID: c63a7f966a05
Revises: cea6c56e4e96
Create Date: 2021-03-18 10:10:08.022654

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'c63a7f966a05'
down_revision = 'cea6c56e4e96'
branch_labels = None
depends_on = None


def upgrade():
    op.add_column('recordings', sa.Column('webhook_uri', sa.VARCHAR(1023), default='', nullable=False))


def downgrade():
    op.drop_column('recordings', 'webhook_uri')
