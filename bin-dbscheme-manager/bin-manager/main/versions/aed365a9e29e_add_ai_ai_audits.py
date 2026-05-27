"""add_ai_ai_audits

Revision ID: aed365a9e29e
Revises: 2929f1291813
Create Date: 2026-05-27 08:01:55.350047

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'aed365a9e29e'
down_revision = '2929f1291813'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        CREATE TABLE IF NOT EXISTS ai_ai_audits (
            id                BINARY(16)    NOT NULL,
            customer_id       BINARY(16)    NOT NULL,
            aicall_id         BINARY(16)    NOT NULL,
            ai_id             BINARY(16)    NOT NULL,
            prompt_history_id BINARY(16)    NOT NULL,
            status            VARCHAR(32)   NOT NULL DEFAULT 'progressing',
            overall_score     INT           NULL,
            evaluation        JSON          NULL,
            language          VARCHAR(32)   NOT NULL DEFAULT 'en-US',
            error             TEXT          NULL,
            tm_create         DATETIME(6)   NOT NULL,
            tm_update         DATETIME(6)   NULL,
            tm_delete         DATETIME(6)   NOT NULL DEFAULT '9999-01-01 00:00:00.000000',
            PRIMARY KEY (id),
            UNIQUE KEY uq_aiaudit_aicall_ai (aicall_id, ai_id),
            INDEX idx_aiaudit_customer_id (customer_id),
            INDEX idx_aiaudit_tm_create (tm_create)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    """)


def downgrade():
    op.execute("DROP TABLE IF EXISTS ai_ai_audits")
