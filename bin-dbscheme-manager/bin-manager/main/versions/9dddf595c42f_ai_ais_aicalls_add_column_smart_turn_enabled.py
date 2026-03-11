"""ai_ais_aicalls_add_column_smart_turn_enabled

Revision ID: 9dddf595c42f
Revises: f3a4b5c6d7e8
Create Date: 2026-03-11 00:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '9dddf595c42f'
down_revision = 'f3a4b5c6d7e8'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE ai_ais ADD smart_turn_enabled TINYINT(1) NOT NULL DEFAULT 0 AFTER vad_config;""")
    op.execute("""ALTER TABLE ai_aicalls ADD ai_smart_turn_enabled TINYINT(1) NOT NULL DEFAULT 0 AFTER ai_vad_config;""")


def downgrade():
    op.execute("""ALTER TABLE ai_ais DROP COLUMN smart_turn_enabled;""")
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN ai_smart_turn_enabled;""")
