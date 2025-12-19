"""ai_aicalls add column ai_aicalls ai_tts_voice_id

Revision ID: bad27b40fe8e
Revises: dfa4f91f18a8
Create Date: 2025-11-01 00:14:12.936764

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'bad27b40fe8e'
down_revision = 'dfa4f91f18a8'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table ai_aicalls add ai_tts_type varchar(255) default '' after ai_engine_data;""")
    op.execute("""update ai_aicalls set ai_tts_type = '';""")

    op.execute("""alter table ai_aicalls add ai_tts_voice_id varchar(255) default '' after ai_tts_type;""")
    op.execute("""update ai_aicalls set ai_tts_voice_id = '';""")
    
    op.execute("""alter table ai_aicalls add ai_stt_type varchar(255) default '' after ai_tts_voice_id;""")
    op.execute("""update ai_aicalls set ai_stt_type = '';""")
    
    op.execute("""alter table ai_aicalls drop column transcribe_id;""")
    op.execute("""alter table ai_aicalls drop column tts_streaming_id;""")
    op.execute("""alter table ai_aicalls drop column tts_streaming_pod_id;""")


    op.execute("""alter table ai_ais add engine_key varchar(255) default '' after engine_data;""")
    op.execute("""update ai_ais set engine_key = '';""")
    
    op.execute("""alter table ai_ais add tts_type varchar(255) default '' after init_prompt;""")
    op.execute("""update ai_ais set tts_type = '';""")
    
    op.execute("""alter table ai_ais add tts_voice_id varchar(255) default '' after tts_type;""")
    op.execute("""update ai_ais set tts_voice_id = '';""")
    
    op.execute("""alter table ai_ais add stt_type varchar(255) default '' after tts_voice_id;""")
    op.execute("""update ai_ais set stt_type = '';""")


def downgrade():
    op.execute("""alter table ai_aicalls drop column ai_tts_type;""")
    op.execute("""alter table ai_aicalls drop column ai_tts_voice_id;""")
    op.execute("""alter table ai_aicalls drop column ai_stt_type;""")
    
    op.execute("""alter table ai_aicalls add transcribe_id binary(16) default '' after confbridge_id;""")
    op.execute("""update ai_aicalls set transcribe_id = unhex(replace('00000000-0000-0000-0000-000000000000', '-', ''));""")
    
    op.execute("""alter table ai_aicalls add tts_streaming_id binary(16) default '' after language;""")
    op.execute("""update ai_aicalls set tts_streaming_id = unhex(replace('00000000-0000-0000-0000-000000000000', '-', ''));""")
    
    op.execute("""alter table ai_aicalls add tts_streaming_pod_id varchar(255) default '' after tts_streaming_id;""")
    op.execute("""update ai_aicalls set tts_streaming_pod_id = '';""")

    op.execute("""alter table ai_ais drop column engine_key;""")
    op.execute("""alter table ai_ais drop column tts_type;""")
    op.execute("""alter table ai_ais drop column tts_voice_id;""")
    op.execute("""alter table ai_ais drop column stt_type;""")
