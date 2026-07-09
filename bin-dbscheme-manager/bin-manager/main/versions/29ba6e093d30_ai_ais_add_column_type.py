"""ai_ais_add_column_type

Revision ID: 29ba6e093d30
Revises: 62f4235bc8e0
Create Date: 2026-07-09 18:04:34.154660

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '29ba6e093d30'
down_revision = '62f4235bc8e0'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE ai_ais ADD `type` VARCHAR(255) NOT NULL DEFAULT 'normal' AFTER detail;""")


def downgrade():
    op.execute("""ALTER TABLE ai_ais DROP COLUMN `type`;""")
