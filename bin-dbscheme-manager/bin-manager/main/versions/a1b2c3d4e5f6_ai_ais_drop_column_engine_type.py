"""ai_ais drop column engine_type

Revision ID: a1b2c3d4e5f6
Revises: 07e99bfda2ef
Create Date: 2026-02-24 16:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'a1b2c3d4e5f6'
down_revision = '07e99bfda2ef'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE ai_ais DROP COLUMN engine_type;""")


def downgrade():
    op.execute("""ALTER TABLE ai_ais ADD engine_type varchar(255) DEFAULT '' AFTER detail;""")
