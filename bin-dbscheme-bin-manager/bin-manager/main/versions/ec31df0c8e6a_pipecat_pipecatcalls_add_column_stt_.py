"""pipecat_pipecatcalls add column stt_language tts_language

Revision ID: ec31df0c8e6a
Revises: f46d9c5c4438
Create Date: 2025-11-30 18:18:39.734307

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ec31df0c8e6a'
down_revision = 'f46d9c5c4438'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE pipecat_pipecatcalls 
        ADD stt_language VARCHAR(255) NOT NULL DEFAULT '' 
        AFTER stt_type;
    """)
    op.execute("""
        UPDATE pipecat_pipecatcalls 
        SET stt_language = '';
    """)

    op.execute("""
        ALTER TABLE pipecat_pipecatcalls 
        ADD tts_language VARCHAR(255) NOT NULL DEFAULT '' 
        AFTER tts_type;
    """)
    op.execute("""
        UPDATE pipecat_pipecatcalls 
        SET tts_language = '';
    """)


def downgrade():
    op.execute("""
        ALTER TABLE pipecat_pipecatcalls 
        DROP COLUMN stt_language;
    """)
    op.execute("""
        ALTER TABLE pipecat_pipecatcalls 
        DROP COLUMN tts_language;
    """)