"""storage_files add column owner_type

Revision ID: b1201e50b736
Revises: 1618235a36a1
Create Date: 2026-01-15 10:53:02.924655

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'b1201e50b736'
down_revision = '1618235a36a1'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE storage_files
        ADD COLUMN owner_type VARCHAR(255) DEFAULT '' AFTER owner_id
    """)


def downgrade():
    op.execute("""
        ALTER TABLE storage_files
        DROP COLUMN owner_type
    """)
