"""call_calls_add_column_metadata

Revision ID: f3a4b5c6d7e8
Revises: e2f3a4b5c6d7
Create Date: 2026-03-09 12:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'f3a4b5c6d7e8'
down_revision = 'e2f3a4b5c6d7'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE call_calls ADD metadata JSON DEFAULT NULL AFTER data;""")


def downgrade():
    op.execute("""ALTER TABLE call_calls DROP COLUMN metadata;""")
