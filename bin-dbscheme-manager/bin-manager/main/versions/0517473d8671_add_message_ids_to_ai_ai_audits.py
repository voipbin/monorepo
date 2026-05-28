"""add_message_ids_to_ai_ai_audits

Revision ID: 0517473d8671
Revises: 072b3191b4a7
Create Date: 2026-05-28 11:56:02.214478

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '0517473d8671'
down_revision = '072b3191b4a7'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE ai_ai_audits
          ADD COLUMN message_ids JSON NULL
          AFTER evaluation
    """)


def downgrade():
    op.execute("""
        ALTER TABLE ai_ai_audits
          DROP COLUMN message_ids
    """)
