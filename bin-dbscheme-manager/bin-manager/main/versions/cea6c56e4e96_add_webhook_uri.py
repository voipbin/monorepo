"""add webhook uri

Revision ID: cea6c56e4e96
Revises: 5d2aab77fd9d
Create Date: 2021-03-11 03:18:28.492161

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'cea6c56e4e96'
down_revision = '5d2aab77fd9d'
branch_labels = None
depends_on = None


def upgrade():
    op.add_column('calls', sa.Column('webhook_uri', sa.VARCHAR(1023)))
    op.add_column('flows', sa.Column('webhook_uri', sa.VARCHAR(1023)))


def downgrade():
    op.drop_column('calls', 'webhook_uri')
    op.drop_column('flows', 'webhook_uri')
