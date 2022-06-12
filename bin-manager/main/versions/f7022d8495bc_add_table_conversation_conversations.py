"""add table conversation_conversations

Revision ID: f7022d8495bc
Revises: 21c5455418a2
Create Date: 2022-06-13 00:37:54.794352

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'f7022d8495bc'
down_revision = '21c5455418a2'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table conversation_conversations(
            -- identity
            id            binary(16),     -- id
            customer_id   binary(16),     -- customer id

            name    varchar(255),
            detail  text,

            reference_type  varchar(255),
            reference_id    varchar(255),

            participants  json,

            -- timestamps
            tm_create   datetime(6),  --
            tm_update   datetime(6),  --
            tm_delete   datetime(6),  --

            primary key(id)
        );
    """)

    op.execute("""
        create index idx_conversation_conversations_customerid on conversation_conversations(customer_id);
    """)

    op.execute("""
        create index idx_conversation_conversations_reference_type_reference_id on conversation_conversations(reference_type, reference_id);
    """)


    op.execute("""
        create table conversation_messages(
            -- identity
            id            binary(16),     -- id
            customer_id   binary(16),     -- customer id

            conversation_id binary(16),

            status varchar(16),

            reference_type  varchar(255),
            reference_id    varchar(255),

            source_target   varchar(255),

            data    blob,

            -- timestamps
            tm_create   datetime(6),  --
            tm_update   datetime(6),  --
            tm_delete   datetime(6),  --

            primary key(id)
        );
    """)

    op.execute("""
        create index idx_conversation_messages_customerid on conversation_messages(customer_id);
    """)

    op.execute("""
        create index idx_conversation_messages_reference_type_reference_id on conversation_messages(reference_type, reference_id);
    """)


def downgrade():
    op.execute("""
        drop table conversation_conversations;
    """)

    op.execute("""
        drop table conversation_messages;
    """)
