"""conversation_accounts_add_column_message_flow_id

Revision ID: e2e30543f2fc
Revises: edc1cab9a95d
Create Date: 2026-03-22 22:03:37.898987

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'e2e30543f2fc'
down_revision = 'edc1cab9a95d'
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.execute(
        "ALTER TABLE conversation_accounts "
        "ADD COLUMN message_flow_id BINARY(16) NOT NULL "
        "DEFAULT (UNHEX(REPLACE('00000000-0000-0000-0000-000000000000', '-', ''))) "
        "AFTER token"
    )


def downgrade() -> None:
    op.execute(
        "ALTER TABLE conversation_accounts DROP COLUMN message_flow_id"
    )
