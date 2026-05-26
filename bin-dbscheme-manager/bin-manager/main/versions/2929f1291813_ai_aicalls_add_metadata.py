"""ai_aicalls_add_metadata

Revision ID: 2929f1291813
Revises: ad3007c85919
Create Date: 2026-05-26 15:07:06.216759

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '2929f1291813'
down_revision = 'ad3007c85919'
branch_labels = None
depends_on = None


def upgrade():
    op.execute(
        "ALTER TABLE ai_aicalls "
        "ADD COLUMN metadata JSON NOT NULL DEFAULT (JSON_OBJECT())"
    )


def downgrade():
    op.execute("ALTER TABLE ai_aicalls DROP COLUMN metadata")
