"""ai_ais_add_current_prompt_history_id

Revision ID: ad3007c85919
Revises: ece6ccf25201
Create Date: 2026-05-26 15:06:36.421330

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ad3007c85919'
down_revision = 'ece6ccf25201'
branch_labels = None
depends_on = None


def upgrade():
    op.execute(
        "ALTER TABLE ai_ais "
        "ADD COLUMN current_prompt_history_id BINARY(16) NOT NULL "
        "DEFAULT 0x00000000000000000000000000000000"
    )


def downgrade():
    op.execute("ALTER TABLE ai_ais DROP COLUMN current_prompt_history_id")
