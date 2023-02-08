"""add table chatbots

Revision ID: 9c150d002975
Revises: ce07efd1486c
Create Date: 2023-02-08 15:00:29.690744

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '9c150d002975'
down_revision = 'ce07efd1486c'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table chatbots(
            id            binary(16),   -- id
            customer_id   binary(16),   -- customer id

            name        varchar(255),   -- name
            detail      text,           -- detail description
            engine_type varchar(255),   --

            tm_create datetime(6),  --
            tm_update datetime(6),  --
            tm_delete datetime(6),  --

            primary key(id)
        );
    """)
    op.execute("""create index idx_chatbots_create on chatbots(tm_create);""")
    op.execute("""create index idx_chatbots_customer_id on chatbots(customer_id);""")


    op.execute("""
        create table chatbotcalls(
            id            binary(16),   -- id
            customer_id   binary(16),   -- customer id

            chatbot_id    binary(16),   -- chatbot id

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
    op.execute("""create index idx_chatbotcalls_customer_id on chatbotcalls(customer_id);""")
    op.execute("""create index idx_chatbotcalls_chatbot_id on chatbotcalls(chatbot_id);""")
    op.execute("""create index idx_chatbotcalls_reference_type on chatbotcalls(reference_type);""")
    op.execute("""create index idx_chatbotcalls_reference_id on chatbotcalls(reference_id);""")
    op.execute("""create index idx_chatbotcalls_transcribe_id on chatbotcalls(transcribe_id);""")
    op.execute("""create index idx_chatbotcalls_create on chatbotcalls(tm_create);""")


def downgrade():
    op.execute("""drop table chatbots;""")
    op.execute("""drop table chatbotcalls;""")
