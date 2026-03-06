"""ai_ais_aicalls_add_column_vad_config

Revision ID: d1a2b3c4d5e6
Revises: c78cf0c45f54
Create Date: 2026-03-06 12:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'd1a2b3c4d5e6'
down_revision = 'c78cf0c45f54'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE ai_ais ADD vad_config JSON DEFAULT NULL AFTER stt_type;""")
    op.execute("""ALTER TABLE ai_aicalls ADD ai_vad_config JSON DEFAULT NULL AFTER ai_stt_type;""")


def downgrade():
    op.execute("""ALTER TABLE ai_ais DROP COLUMN vad_config;""")
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN ai_vad_config;""")
