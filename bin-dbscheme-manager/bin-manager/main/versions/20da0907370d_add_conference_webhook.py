"""add conference webhook

Revision ID: 20da0907370d
Revises: 9443d2d65ad8
Create Date: 2021-09-28 10:42:43.418648

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '20da0907370d'
down_revision = '9443d2d65ad8'
branch_labels = None
depends_on = None


def upgrade():
    op.add_column('conferences', sa.Column('webhook_uri', sa.VARCHAR(1023), default='', nullable=False))


def downgrade():
    op.drop_column('conferences', 'webhook_uri')
