"""add_ai_ai_prompt_histories

Revision ID: ece6ccf25201
Revises: 1eef6c730e85
Create Date: 2026-05-22 15:45:43.096255

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ece6ccf25201'
down_revision = '1eef6c730e85'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        CREATE TABLE IF NOT EXISTS ai_ai_prompt_histories (
            -- Append-only: no tm_update or tm_delete (rows are immutable history)
            id          BINARY(16)   NOT NULL,
            customer_id BINARY(16)   NOT NULL,
            ai_id       BINARY(16)   NOT NULL,
            prompt      LONGTEXT     NOT NULL DEFAULT '',
            tm_create   DATETIME(6)  NOT NULL,
            PRIMARY KEY (id),
            INDEX idx_ai_ai_prompt_histories_ai_id_tm_create (ai_id, tm_create),
            INDEX idx_ai_ai_prompt_histories_customer_id (customer_id)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    """)


def downgrade():
    op.execute("DROP TABLE IF EXISTS ai_ai_prompt_histories")
