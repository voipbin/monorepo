"""ai_aicall_participants_create_table

Revision ID: 1eef6c730e85
Revises: ef130bcf8401
Create Date: 2026-05-22 11:38:01.620425

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = '1eef6c730e85'
down_revision = 'ef130bcf8401'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        CREATE TABLE ai_aicall_participants (
            ai_id      BINARY(16) NOT NULL,
            aicall_id  BINARY(16) NOT NULL,
            tm_create  DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
            PRIMARY KEY (ai_id, aicall_id),
            INDEX idx_ai_aicall_participants_aicall_id (aicall_id)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    """)


def downgrade():
    op.execute("DROP TABLE IF EXISTS ai_aicall_participants")
