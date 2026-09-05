"""ai_messages_add_column_origin

Revision ID: 42cf2ffc4273
Revises: 6e848e2cf613
Create Date: 2026-09-05 00:17:44.061583

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '42cf2ffc4273'
down_revision = '6e848e2cf613'
branch_labels = None
depends_on = None


def upgrade():
    # origin distinguishes a message the AI produced on its own initiative from
    # one that answers or asks something (docs/plans/
    # 2026-09-03-insight-ai-realtime-listen-design.md §5.6.2).
    #
    # Three values in use:
    #   ''                -- OriginNone, the default: every ordinary message.
    #   'proactive'       -- an AI-initiated notification the agent should see;
    #                        the frontends badge it, and it IS replayed into
    #                        future LLM context so the AI remembers what it said.
    #   'listen_internal' -- mechanical tool-call/tool-result rows written during
    #                        a listen evaluation turn; excluded from every future
    #                        LLM replay so they cannot evict the system prompt or
    #                        the agent's own Q&A history.
    #
    # varchar(16) NOT NULL DEFAULT '' matches the role/direction columns' shape
    # on this table, and makes every pre-existing row read back as OriginNone
    # with no backfill.
    op.execute("""
        ALTER TABLE ai_messages
          ADD COLUMN origin VARCHAR(16) NOT NULL DEFAULT ''
    """)


def downgrade():
    op.execute("""
        ALTER TABLE ai_messages
          DROP COLUMN origin
    """)
