"""ai_teams add parameter column and ai_aicalls add team_parameter column

Revision ID: f1a2b3c4d5e6
Revises: ed4cff99a82e
Create Date: 2026-02-28 01:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'f1a2b3c4d5e6'
down_revision = 'ed4cff99a82e'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE ai_teams ADD parameter JSON AFTER members;""")
    op.execute("""ALTER TABLE ai_aicalls ADD team_parameter JSON AFTER ai_stt_type;""")


def downgrade():
    op.execute("""ALTER TABLE ai_aicalls DROP COLUMN team_parameter;""")
    op.execute("""ALTER TABLE ai_teams DROP COLUMN parameter;""")
