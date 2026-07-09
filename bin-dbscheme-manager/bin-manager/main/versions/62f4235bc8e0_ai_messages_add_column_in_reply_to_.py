"""ai_messages_add_column_in_reply_to_message_id

Revision ID: 62f4235bc8e0
Revises: a5a40c93d3e6
Create Date: 2026-07-09

Adds in_reply_to_message_id to ai_messages (VOIP-1234 subtask 2). Correlates
an assistant message with the user-authored message it answers, so agent-
facing clients can disambiguate responses when an AIcall is reused for a
rapid sequence of sends (e.g. a second question asked before the first bot
response arrives). Zero UUID (the existing zero-UUID convention used
elsewhere in this schema, e.g. ai_aicalls.active_reference_key) for rows with
no correlation available (user messages, and legacy/voice paths that predate
this field).
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '62f4235bc8e0'
down_revision = 'a5a40c93d3e6'
branch_labels = None
depends_on = None


def upgrade():
    op.add_column('ai_messages',
        sa.Column('in_reply_to_message_id', sa.BINARY(16), nullable=False,
                  server_default=sa.text("0x00000000000000000000000000000000")))


def downgrade():
    op.drop_column('ai_messages', 'in_reply_to_message_id')
