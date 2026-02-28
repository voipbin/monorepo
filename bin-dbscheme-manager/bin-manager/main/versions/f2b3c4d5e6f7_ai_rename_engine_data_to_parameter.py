"""ai_ais rename engine_data to parameter, ai_aicalls merge ai_engine_data + team_parameter into parameter

Revision ID: f2b3c4d5e6f7
Revises: f1b2c3d4e5f6
Create Date: 2026-02-28 02:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'f2b3c4d5e6f7'
down_revision = 'f1b2c3d4e5f6'
branch_labels = None
depends_on = None


def upgrade():
    # Rename engine_data -> parameter in ai_ais
    op.execute("""ALTER TABLE ai_ais CHANGE engine_data parameter JSON;""")

    # Merge ai_engine_data + team_parameter -> parameter in ai_aicalls
    op.execute("""ALTER TABLE ai_aicalls ADD parameter JSON AFTER ai_stt_type;""")
    op.execute("""UPDATE ai_aicalls SET parameter = ai_engine_data;""")
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN ai_engine_data;""")
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN team_parameter;""")


def downgrade():
    # Restore ai_engine_data and team_parameter in ai_aicalls
    op.execute("""ALTER TABLE ai_aicalls ADD ai_engine_data JSON AFTER ai_engine_model;""")
    op.execute("""ALTER TABLE ai_aicalls ADD team_parameter JSON AFTER ai_stt_type;""")
    op.execute("""UPDATE ai_aicalls SET ai_engine_data = parameter;""")
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN parameter;""")

    # Restore engine_data in ai_ais
    op.execute("""ALTER TABLE ai_ais CHANGE parameter engine_data JSON;""")
