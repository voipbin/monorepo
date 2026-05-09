"""route_providers_add_column_codecs

Revision ID: a3c811eb1d4c
Revises: ff02d243a799
Create Date: 2026-05-09 01:29:03.088981

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'a3c811eb1d4c'
down_revision = 'ff02d243a799'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE route_providers
        ADD COLUMN codecs VARCHAR(255) NOT NULL DEFAULT ''
    """)


def downgrade():
    op.execute("""
        ALTER TABLE route_providers
        DROP COLUMN codecs
    """)
