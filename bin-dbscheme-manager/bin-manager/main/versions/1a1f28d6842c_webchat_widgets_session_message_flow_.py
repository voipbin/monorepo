"""webchat_widgets_session_message_flow_split

Revision ID: 1a1f28d6842c
Revises: 4dd9760302b8
Create Date: 2026-07-17 23:55:58.572983

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '1a1f28d6842c'
down_revision = '4dd9760302b8'
branch_labels = None
depends_on = None


def upgrade():
    # flow_id -> session_flow_id (breaking rename: the widget's Flow
    # now fires at Session creation time, not on the visitor's first
    # inbound message -- see design doc
    # 2026-07-17-webchat-widget-session-message-flow-split-design.md).
    # This is a same-day, unreleased feature (webchat-manager merged
    # 2026-07-16/17, no external consumers yet), so a breaking rename
    # is safe.
    op.execute("""
        alter table webchat_widgets
        change column flow_id session_flow_id binary(16);
    """)

    # message_flow_id: optional, independent Flow that fires on EVERY
    # inbound message (mirrors bin-conversation-manager's
    # Account.MessageFlowID/Number.MessageFlowID pattern for
    # LINE/WhatsApp/SMS).
    op.execute("""
        alter table webchat_widgets
        add column message_flow_id binary(16) after session_flow_id;
    """)


def downgrade():
    op.execute("""
        alter table webchat_widgets
        drop column message_flow_id;
    """)

    op.execute("""
        alter table webchat_widgets
        change column session_flow_id flow_id binary(16);
    """)
