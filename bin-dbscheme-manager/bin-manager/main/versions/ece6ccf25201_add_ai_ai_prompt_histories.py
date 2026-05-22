"""add_ai_ai_prompt_histories

Revision ID: ece6ccf25201
Revises: ef130bcf8401
Create Date: 2026-05-22 15:45:43.096255

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ece6ccf25201'
down_revision = 'ef130bcf8401'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        CREATE TABLE ai_ai_prompt_histories (
            id          BINARY(16)   NOT NULL,
            customer_id BINARY(16)   NOT NULL,
            ai_id       BINARY(16)   NOT NULL,
            prompt      LONGTEXT     NOT NULL DEFAULT '',
            tm_create   DATETIME(6)  NOT NULL,
            PRIMARY KEY (id),
            KEY idx_ai_ai_prompt_histories_ai_id (ai_id),
            KEY idx_ai_ai_prompt_histories_customer_id (customer_id)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    """)


def downgrade():
    op.execute("DROP TABLE IF EXISTS ai_ai_prompt_histories")
