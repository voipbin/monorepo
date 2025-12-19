"""ai_aicalls add tts_streaming
Revision ID: 6efdc5294d5e
Revises: 6c88940e2936
Create Date: 2025-07-08 12:32:15.938429
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '6efdc5294d5e'
down_revision = '6c88940e2936'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table ai_aicalls add column tts_streaming_id binary(16) after language;""")
    op.execute("""update ai_aicalls set tts_streaming_id = unhex(replace('00000000-0000-0000-0000-000000000000', '-', ''));""")

    op.execute("""alter table ai_aicalls add column tts_streaming_pod_id varchar(255) after tts_streaming_id;""")
    op.execute("""update ai_aicalls set tts_streaming_pod_id = '';""")


def downgrade():
    op.execute("""alter table ai_aicalls drop column tts_streaming_id;""")
    op.execute("""alter table ai_aicalls drop column tts_streaming_pod_id;""")
