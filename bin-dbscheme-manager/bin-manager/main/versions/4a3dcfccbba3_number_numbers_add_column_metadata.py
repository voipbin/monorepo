"""number_numbers add column metadata

Revision ID: 4a3dcfccbba3
Revises: 9dddf595c42f
Create Date: 2026-03-12 00:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '4a3dcfccbba3'
down_revision = '9dddf595c42f'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE number_numbers ADD metadata JSON DEFAULT NULL AFTER emergency_enabled;""")


def downgrade():
    op.execute("""ALTER TABLE number_numbers DROP COLUMN metadata;""")
