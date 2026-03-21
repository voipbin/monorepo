"""ai_aicalls add column current_member_id

Revision ID: edc1cab9a95d
Revises: 1f362e55653b
Create Date: 2026-03-22 00:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'edc1cab9a95d'
down_revision = '1f362e55653b'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("ALTER TABLE ai_aicalls ADD COLUMN current_member_id binary(16) AFTER pipecatcall_id")


def downgrade():
    op.execute("ALTER TABLE ai_aicalls DROP COLUMN current_member_id")
