"""chatbot_messages create table

Revision ID: 403c1ab3552c
Revises: 4723f143cfe1
Create Date: 2025-02-25 00:23:19.365729

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '403c1ab3552c'
down_revision = '4723f143cfe1'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table chatbot_messages(
        -- identity
        id            binary(16),     -- id
        customer_id   binary(16),     -- customer id
        chatbotcall_id binary(16),    -- chatbotcall id

        direction     varchar(255),   -- direction
        role          varchar(255),   -- role
        content       text,           -- content

        -- timestamps
        tm_create datetime(6),  --
        tm_delete datetime(6),

        primary key(id)
        );
    """)
    op.execute("""create index idx_chatbot_messages_create on chatbot_messages(tm_create);""")
    op.execute("""create index idx_chatbot_messages_customer_id on chatbot_messages(customer_id);""")
    op.execute("""create index idx_chatbot_messages_chatbotcall_id on chatbot_messages(chatbotcall_id);""")


def downgrade():
    op.execute("""drop table chatbot_messages;""")
