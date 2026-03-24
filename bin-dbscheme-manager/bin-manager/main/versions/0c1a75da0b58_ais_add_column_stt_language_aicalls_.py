"""ais add column stt_language aicalls rename column language to stt_language

Revision ID: 0c1a75da0b58
Revises: e2e30543f2fc
Create Date: 2026-03-24 05:11:29.434631

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '0c1a75da0b58'
down_revision = 'e2e30543f2fc'
branch_labels = None
depends_on = None


def upgrade():
    # Add stt_language to ais table
    op.execute("""ALTER TABLE ai_ais ADD COLUMN stt_language VARCHAR(255) NOT NULL DEFAULT '' AFTER stt_type""")

    # Rename language -> stt_language in aicalls table
    op.execute("""ALTER TABLE ai_aicalls CHANGE COLUMN language stt_language VARCHAR(255) NOT NULL DEFAULT ''""")


def downgrade():
    # Rename stt_language -> language in aicalls table
    op.execute("""ALTER TABLE ai_aicalls CHANGE COLUMN stt_language language VARCHAR(255) NOT NULL DEFAULT ''""")

    # Remove stt_language from ais table
    op.execute("""ALTER TABLE ai_ais DROP COLUMN stt_language""")
