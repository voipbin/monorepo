"""chat_drop_tables_chat_chats_chat_chatrooms_chat_messagechats_chat_messagechatrooms

Revision ID: 5b7401675bcc
Revises: 91c93611e3ec
Create Date: 2026-01-30 11:02:03.028410

Drops all tables related to the deprecated bin-chat-manager service.
The chat functionality has been replaced by bin-talk-manager.

Tables dropped:
- chat_chats
- chat_chatrooms
- chat_messagechats
- chat_messagechatrooms
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '5b7401675bcc'
down_revision = '91c93611e3ec'
branch_labels = None
depends_on = None


def upgrade():
    # Drop indexes first (they are automatically dropped with the table in MySQL,
    # but being explicit for clarity)

    # Drop chat_messagechatrooms table
    op.execute("""DROP TABLE IF EXISTS chat_messagechatrooms;""")

    # Drop chat_messagechats table
    op.execute("""DROP TABLE IF EXISTS chat_messagechats;""")

    # Drop chat_chatrooms table
    op.execute("""DROP TABLE IF EXISTS chat_chatrooms;""")

    # Drop chat_chats table
    op.execute("""DROP TABLE IF EXISTS chat_chats;""")


def downgrade():
    # WARNING: This downgrade only recreates empty tables.
    # Data cannot be recovered once dropped.

    # Recreate chat_chats
    op.execute("""
        CREATE TABLE chat_chats (
            id          BINARY(16),
            customer_id BINARY(16),
            type        VARCHAR(255),
            room_owner_id BINARY(16),
            participant_ids JSON,
            name        VARCHAR(255),
            detail      TEXT,
            tm_create   DATETIME(6),
            tm_update   DATETIME(6),
            tm_delete   DATETIME(6),
            PRIMARY KEY(id)
        );
    """)
    op.execute("""CREATE INDEX idx_chat_chats_customer_id ON chat_chats(customer_id);""")
    op.execute("""CREATE INDEX idx_chat_chats_room_owner_id ON chat_chats(room_owner_id);""")

    # Recreate chat_chatrooms
    op.execute("""
        CREATE TABLE chat_chatrooms (
            id          BINARY(16),
            customer_id BINARY(16),
            owner_type  VARCHAR(255),
            type        VARCHAR(255),
            chat_id     BINARY(16),
            owner_id    BINARY(16),
            room_owner_id BINARY(16),
            participant_ids JSON,
            name        VARCHAR(255),
            detail      TEXT,
            tm_create   DATETIME(6),
            tm_update   DATETIME(6),
            tm_delete   DATETIME(6),
            PRIMARY KEY(id)
        );
    """)
    op.execute("""CREATE INDEX idx_chat_chatrooms_customer_id ON chat_chatrooms(customer_id);""")
    op.execute("""CREATE INDEX idx_chat_chatrooms_chat_id ON chat_chatrooms(chat_id);""")
    op.execute("""CREATE INDEX idx_chat_chatrooms_owner_id ON chat_chatrooms(owner_id);""")
    op.execute("""CREATE INDEX idx_chat_chatrooms_chat_id_owner_id ON chat_chatrooms(chat_id, owner_id);""")
    op.execute("""CREATE INDEX idx_chat_chatrooms_room_owner_id ON chat_chatrooms(room_owner_id);""")

    # Recreate chat_messagechats
    op.execute("""
        CREATE TABLE chat_messagechats (
            id          BINARY(16),
            customer_id BINARY(16),
            chat_id     BINARY(16),
            source      JSON,
            type        VARCHAR(255),
            text        TEXT,
            medias      JSON,
            tm_create   DATETIME(6),
            tm_update   DATETIME(6),
            tm_delete   DATETIME(6),
            PRIMARY KEY(id)
        );
    """)
    op.execute("""CREATE INDEX idx_chat_messagechats_customer_id ON chat_messagechats(customer_id);""")
    op.execute("""CREATE INDEX idx_chat_messagechats_chat_id ON chat_messagechats(chat_id);""")

    # Recreate chat_messagechatrooms
    op.execute("""
        CREATE TABLE chat_messagechatrooms (
            id             BINARY(16),
            customer_id    BINARY(16),
            owner_type     VARCHAR(255),
            owner_id       BINARY(16),
            chatroom_id    BINARY(16),
            messagechat_id BINARY(16),
            source         JSON,
            type           VARCHAR(255),
            text           TEXT,
            medias         JSON,
            tm_create      DATETIME(6),
            tm_update      DATETIME(6),
            tm_delete      DATETIME(6),
            PRIMARY KEY(id)
        );
    """)
    op.execute("""CREATE INDEX idx_chat_messagechatrooms_customer_id ON chat_messagechatrooms(customer_id);""")
    op.execute("""CREATE INDEX idx_chat_messagechatrooms_chatroom_id ON chat_messagechatrooms(chatroom_id);""")
    op.execute("""CREATE INDEX idx_chat_messagechatrooms_messagechat_id ON chat_messagechatrooms(messagechat_id);""")
