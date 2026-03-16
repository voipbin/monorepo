"""ai_messages add column activeflow_id

Revision ID: 889f8fc50634
Revises: fd2a3b4c5d6e
Create Date: 2026-03-15 12:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = '889f8fc50634'
down_revision = 'fd2a3b4c5d6e'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE ai_messages
        ADD COLUMN activeflow_id binary(16) AFTER aicall_id
    """)
    op.execute("""
        CREATE INDEX idx_ai_messages_activeflow_id ON ai_messages(activeflow_id)
    """)


def downgrade():
    op.execute("""
        DROP INDEX idx_ai_messages_activeflow_id ON ai_messages
    """)
    op.execute("""
        ALTER TABLE ai_messages
        DROP COLUMN activeflow_id
    """)
