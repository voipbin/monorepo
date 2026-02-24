"""ai_aicalls drop column ai_engine_type

Revision ID: 07e99bfda2ef
Revises: fe1a2b3c4d5e
Create Date: 2026-02-24 15:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '07e99bfda2ef'
down_revision = 'fe1a2b3c4d5e'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN ai_engine_type;""")


def downgrade():
    op.execute("""ALTER TABLE ai_aicalls ADD ai_engine_type varchar(255) DEFAULT '' AFTER ai_id;""")
