"""add table chats

Revision ID: 0eb33809a60c
Revises: e6990d5b510c
Create Date: 2022-09-16 23:20:29.412200

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '0eb33809a60c'
down_revision = 'e6990d5b510c'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table chats(
            -- identity
            id          binary(16),
            customer_id binary(16),

            type        varchar(255),

            owner_id        binary(16),
            participant_ids json,

            name      varchar(255),
            detail    text,

            -- timestamps
            tm_create datetime(6),  -- create
            tm_update datetime(6),  -- update
            tm_delete datetime(6),  -- delete

            primary key(id)
        );
    """)
    op.execute("""create index idx_chats_customer_id on chats(customer_id);""")
    op.execute("""create index idx_chats_owner_id on chats(owner_id);""")

    op.execute("""
        create table chatrooms(
            -- identity
            id          binary(16),
            customer_id binary(16),

            type        varchar(255),
            chat_id     binary(16),

            owner_id          binary(16),
            participant_ids   json,

            name      varchar(255),
            detail    text,

            -- timestamps
            tm_create datetime(6),  -- create
            tm_update datetime(6),  -- update
            tm_delete datetime(6),  -- delete

            primary key(id)
        );
    """)
    op.execute("""create index idx_chatrooms_customer_id on chatrooms(customer_id);""")
    op.execute("""create index idx_chatrooms_chat_id on chatrooms(chat_id);""")
    op.execute("""create index idx_chatrooms_owner_id on chatrooms(owner_id);""")
    op.execute("""create index idx_chatrooms_chat_id_owner_id on chatrooms(chat_id, owner_id);""")

    op.execute("""
        create table messagechats(
            -- identity
            id          binary(16),
            customer_id binary(16),

            chat_id binary(16),

            source  json,
            type    varchar(255),
            text    text,
            medias  json,

            -- timestamps
            tm_create datetime(6),  -- create
            tm_update datetime(6),  -- update
            tm_delete datetime(6),  -- delete

            primary key(id)
        );
    """)
    op.execute("""create index idx_messagechats_customer_id on messagechats(customer_id);""")
    op.execute("""create index idx_messagechats_chat_id on messagechats(chat_id);""")

    op.execute("""
        create table messagechatrooms(
            -- identity
            id          binary(16),
            customer_id binary(16),

            chatroom_id binary(16),
            messagechat_id binary(16),

            source  json,
            type    varchar(255),
            text    text,
            medias  json,

            -- timestamps
            tm_create datetime(6),  -- create
            tm_update datetime(6),  -- update
            tm_delete datetime(6),  -- delete

            primary key(id)
        );
    """)
    op.execute("""create index idx_messagechatrooms_customer_id on messagechatrooms(customer_id);""")
    op.execute("""create index idx_messagechatrooms_chatroom_id on messagechatrooms(chatroom_id);""")
    op.execute("""create index idx_messagechatrooms_messagechat_id on messagechatrooms(messagechat_id);""")


def downgrade():
    op.execute("""drop table chats;""")
    op.execute("""drop table chatrooms;""")
    op.execute("""drop table messagechats;""")
    op.execute("""drop table messagechatrooms;""")

