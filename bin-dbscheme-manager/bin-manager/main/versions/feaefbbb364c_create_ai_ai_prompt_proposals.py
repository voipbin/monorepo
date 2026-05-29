"""create_ai_ai_prompt_proposals

Revision ID: feaefbbb364c
Revises: 0517473d8671
Create Date: 2026-05-29 11:57:08.512868

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'feaefbbb364c'
down_revision = '0517473d8671'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        CREATE TABLE IF NOT EXISTS ai_ai_prompt_proposals (
            id                         BINARY(16)   NOT NULL,
            customer_id                BINARY(16)   NOT NULL,
            ai_id                      BINARY(16)   NOT NULL,
            audit_ids                  JSON         NOT NULL,
            basis_prompt_history_id    BINARY(16)   NOT NULL,
            original_prompt            TEXT         NULL,
            proposed_prompt            TEXT         NULL,
            rationale                  TEXT         NULL,
            status                     VARCHAR(32)  NOT NULL DEFAULT 'progressing',
            error                      VARCHAR(128) NOT NULL DEFAULT '',
            applied_prompt_history_id  BINARY(16)   NOT NULL DEFAULT 0x00000000000000000000000000000000,
            tm_create                  DATETIME(6)  NOT NULL,
            tm_update                  DATETIME(6)  NULL DEFAULT NULL,
            tm_delete                  DATETIME(6)  NULL DEFAULT NULL,
            PRIMARY KEY (id),
            INDEX idx_aipromptproposal_customer_ai_create (customer_id, ai_id, tm_create),
            INDEX idx_aipromptproposal_ai_status          (ai_id, status, tm_delete),
            INDEX idx_aipromptproposal_customer_status    (customer_id, status, tm_create)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    """)


def downgrade():
    op.execute("DROP TABLE IF EXISTS ai_ai_prompt_proposals")
