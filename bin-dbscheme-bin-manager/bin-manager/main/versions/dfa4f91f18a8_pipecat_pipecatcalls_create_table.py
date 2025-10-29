"""pipecat_pipecatcalls create table

Revision ID: dfa4f91f18a8
Revises: 069e6bf3ccfa
Create Date: 2025-10-30 06:52:24.779650

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'dfa4f91f18a8'
down_revision = '069e6bf3ccfa'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table pipecat_pipecatcalls(
            -- identity
            id            binary(16),   -- id
            customer_id   binary(16),   -- customer id

            activeflow_id     binary(16),
            reference_type    varchar(255),
            reference_id      binary(16),

            host_id varchar(255),  -- host id

            llm_type      varchar(255),
            llm_messages  json,
            stt_type      varchar(255),
            tts_type      varchar(255),
            tts_voice_id  varchar(255),

            -- timestamps
            tm_create datetime(6),  --
            tm_update datetime(6),  --
            tm_delete datetime(6),  --

            primary key(id)
        );
        """)
    op.execute("""create index idx_pipecat_pipecatcalls_customer_id on pipecat_pipecatcalls(customer_id);""")
    op.execute("""create index idx_pipecat_pipecatcalls_activeflow_id on pipecat_pipecatcalls(activeflow_id);""")
    op.execute("""create index idx_pipecat_pipecatcalls_reference_type on pipecat_pipecatcalls(reference_type);""")
    op.execute("""create index idx_pipecat_pipecatcalls_reference_id on pipecat_pipecatcalls(reference_id);""")
    op.execute("""create index idx_pipecat_pipecatcalls_host_id on pipecat_pipecatcalls(host_id);""")
    op.execute("""create index idx_pipecat_pipecatcalls_tm_create on pipecat_pipecatcalls(tm_create);""")


def downgrade():
    op.execute("""drop table pipecat_pipecatcalls;""")
