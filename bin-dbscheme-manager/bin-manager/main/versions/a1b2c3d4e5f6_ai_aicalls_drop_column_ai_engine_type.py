"""ai_aicalls drop column ai_engine_type

Revision ID: a1b2c3d4e5f6
Revises: fe1a2b3c4d5e
Create Date: 2026-02-24 15:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'a1b2c3d4e5f6'
down_revision = 'fe1a2b3c4d5e'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN ai_engine_type;""")


def downgrade():
    op.execute("""ALTER TABLE ai_aicalls ADD ai_engine_type varchar(255) DEFAULT '' AFTER ai_id;""")
