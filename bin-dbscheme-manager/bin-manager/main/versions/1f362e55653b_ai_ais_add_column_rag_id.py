"""ai_ais add column rag_id

Revision ID: 1f362e55653b
Revises: g1a2b3c4d5e6
Create Date: 2026-03-20 21:50:45.393338

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '1f362e55653b'
down_revision = 'g1a2b3c4d5e6'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("ALTER TABLE ai_ais ADD COLUMN rag_id binary(16) AFTER engine_key")


def downgrade():
    op.execute("ALTER TABLE ai_ais DROP COLUMN rag_id")
