"""talk create tables talk_chats talk_messages talk_participants

Revision ID: f6d7d64e259e
Revises: b1201e50b736
Create Date: 2026-01-17 03:15:01.368277

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'f6d7d64e259e'
down_revision = 'b1201e50b736'
branch_labels = None
depends_on = None


def upgrade():
    # Create talk_chats table
    op.execute("""
        CREATE TABLE talk_chats (
            id              BINARY(16) PRIMARY KEY,
            customer_id     BINARY(16) NOT NULL,
            type            VARCHAR(255) NOT NULL,
            tm_create       DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
            tm_update       DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
            tm_delete       DATETIME(6) NULL,
            INDEX idx_customer_id (customer_id)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    """)

    # Create talk_participants table
    op.execute("""
        CREATE TABLE talk_participants (
            id              BINARY(16) PRIMARY KEY,
            chat_id         BINARY(16) NOT NULL,
            owner_type      VARCHAR(255) NOT NULL,
            owner_id        BINARY(16) NOT NULL,
            tm_joined       DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
            INDEX idx_chat_id (chat_id),
            INDEX idx_owner (owner_type, owner_id),
            UNIQUE KEY unique_participant (chat_id, owner_type, owner_id),
            FOREIGN KEY (chat_id) REFERENCES talk_chats(id) ON DELETE CASCADE
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    """)

    # Create talk_messages table
    op.execute("""
        CREATE TABLE talk_messages (
            id              BINARY(16) PRIMARY KEY,
            customer_id     BINARY(16) NOT NULL,
            chat_id         BINARY(16) NOT NULL,
            parent_id       BINARY(16) NULL,
            owner_type      VARCHAR(255) NOT NULL,
            owner_id        BINARY(16) NOT NULL,
            type            VARCHAR(255) NOT NULL,
            text            TEXT,
            medias          JSON,
            metadata        JSON,
            tm_create       DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
            tm_update       DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
            tm_delete       DATETIME(6) NULL,
            INDEX idx_chat_id (chat_id),
            INDEX idx_parent_id (parent_id),
            INDEX idx_customer_id (customer_id),
            INDEX idx_owner (owner_type, owner_id),
            FOREIGN KEY (chat_id) REFERENCES talk_chats(id) ON DELETE CASCADE,
            FOREIGN KEY (parent_id) REFERENCES talk_messages(id) ON DELETE SET NULL
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    """)


def downgrade():
    op.execute("DROP TABLE IF EXISTS talk_messages")
    op.execute("DROP TABLE IF EXISTS talk_participants")
    op.execute("DROP TABLE IF EXISTS talk_chats")
