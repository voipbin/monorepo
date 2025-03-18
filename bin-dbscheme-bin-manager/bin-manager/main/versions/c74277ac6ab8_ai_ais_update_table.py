"""ai_ais update table

Revision ID: c74277ac6ab8
Revises: 1c0bc3259406
Create Date: 2025-03-18 02:25:15.315174

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'c74277ac6ab8'
down_revision = '1c0bc3259406'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table ai_aicalls(
            id            binary(16),   -- id
            customer_id   binary(16),   -- customer id

            ai_id            binary(16),   -- ai id
            ai_engine_type   varchar(255), -- ai engine type
            ai_engine_model  varchar(255),
            ai_engine_data   json,

            activeflow_id     binary(16),
            reference_type    varchar(255),
            reference_id      binary(16),

            confbridge_id     binary(16),
            transcribe_id     binary(16),

            status  varchar(255),   -- status

            gender    varchar(255), -- gender
            language  varchar(255), -- language

            tm_end    datetime(6),  --
            tm_create datetime(6),  --
            tm_update datetime(6),  --
            tm_delete datetime(6),  --

            primary key(id)
        );
    """)
    op.execute("""create index idx_ai_aicalls_customer_id on ai_aicalls(customer_id);""")
    op.execute("""create index idx_ai_aicalls_ai_id on ai_aicalls(ai_id);""")
    op.execute("""create index idx_ai_aicalls_reference_type on ai_aicalls(reference_type);""")
    op.execute("""create index idx_ai_aicalls_reference_id on ai_aicalls(reference_id);""")
    op.execute("""create index idx_ai_aicalls_transcribe_id on ai_aicalls(transcribe_id);""")
    op.execute("""create index idx_ai_aicalls_create on ai_aicalls(tm_create);""")
    op.execute("""create index idx_ai_aicalls_activeflow_id on ai_aicalls(activeflow_id);""")
    op.execute("""
        INSERT INTO ai_aicalls (
            id, customer_id, ai_id, ai_engine_type, ai_engine_model, ai_engine_data,
            activeflow_id, reference_type, reference_id,
            confbridge_id, transcribe_id, status, gender, language,
            tm_end, tm_create, tm_update, tm_delete
        )
        SELECT 
            id, customer_id, chatbot_id, chatbot_engine_type, chatbot_engine_model, chatbot_engine_data,
            activeflow_id, reference_type, reference_id,
            confbridge_id, transcribe_id, status, gender, language,
            tm_end, tm_create, tm_update, tm_delete
        FROM chatbot_chatbotcalls;
    """)


    op.execute("""
        create table ai_ais(
            id            binary(16),   -- id
            customer_id   binary(16),   -- customer id

            name        varchar(255),   -- name
            detail      text,           -- detail description

            engine_type   varchar(255),   --
            engine_model  varchar(255),
            engine_data   json,

            init_prompt   text,           -- initial prompt

            tm_create datetime(6),  --
            tm_update datetime(6),  --
            tm_delete datetime(6),  --

            primary key(id)
        );
    """)
    op.execute("""create index idx_ai_ais_create on ai_ais(tm_create);""")
    op.execute("""create index idx_ai_ais_customer_id on ai_ais(customer_id);""")
    op.execute("""
        INSERT INTO ai_ais (
            id, customer_id, name, detail,
            engine_type, engine_model, engine_data,
            init_prompt, tm_create, tm_update, tm_delete
        )
        SELECT
            id, customer_id, name, detail,
            engine_type, engine_model, engine_data,
            init_prompt, tm_create, tm_update, tm_delete
        FROM chatbot_chatbots;
    """)


    op.execute("""
        create table ai_messages(
            -- identity
            id            binary(16),     -- id
            customer_id   binary(16),     -- customer id
            aicall_id binary(16),

            direction     varchar(255),   -- direction
            role          varchar(255),   -- role
            content       text,           -- content

            -- timestamps
            tm_create datetime(6),  --
            tm_delete datetime(6),

            primary key(id)
        );
    """)
    op.execute("""create index idx_ai_messages_create on ai_messages(tm_create);""")
    op.execute("""create index idx_ai_messages_customer_id on ai_messages(customer_id);""")
    op.execute("""create index idx_ai_messages_aicall_id on ai_messages(aicall_id);""")
    op.execute("""
        INSERT INTO ai_messages (
            id, customer_id, aicall_id, direction, role, content,
            tm_create, tm_delete
        )
        SELECT
            id, customer_id, chatbotcall_id, direction, role, content,
            tm_create, tm_delete
        FROM chatbot_messages;
    """)


    op.execute("""drop table chatbot_chatbotcalls;""")
    op.execute("""drop table chatbot_chatbots;""")
    op.execute("""drop table chatbot_messages;""")


def downgrade():
    op.execute("""drop table ai_aicalls;""")
    op.execute("""drop table ai_ais;""")
    op.execute("""drop table ai_messages;""")
