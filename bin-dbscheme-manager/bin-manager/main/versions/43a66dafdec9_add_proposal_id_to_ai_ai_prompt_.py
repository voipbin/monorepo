"""add_proposal_id_to_ai_ai_prompt_histories

Revision ID: 43a66dafdec9
Revises: feaefbbb364c
Create Date: 2026-05-29 12:00:32.784036

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '43a66dafdec9'
down_revision = 'feaefbbb364c'
branch_labels = None
depends_on = None


def upgrade():
    op.execute(
        "ALTER TABLE ai_ai_prompt_histories "
        "ADD COLUMN proposal_id BINARY(16) NOT NULL "
        "DEFAULT 0x00000000000000000000000000000000, "
        "ALGORITHM=INSTANT, LOCK=NONE"
    )


def downgrade():
    op.execute("ALTER TABLE ai_ai_prompt_histories DROP COLUMN proposal_id")
