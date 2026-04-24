"""route_providers add column metadata

Revision ID: 629a307b92d0
Revises: aa0d0a29625e
Create Date: 2026-04-24 11:35:15.865735

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '629a307b92d0'
down_revision = 'aa0d0a29625e'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE route_providers
        ADD COLUMN metadata JSON
    """)


def downgrade():
    op.execute("""
        ALTER TABLE route_providers
        DROP COLUMN metadata
    """)
