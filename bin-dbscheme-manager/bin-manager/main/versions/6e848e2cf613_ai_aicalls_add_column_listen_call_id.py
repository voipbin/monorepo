"""ai_aicalls_add_column_listen_call_id

Revision ID: 6e848e2cf613
Revises: ab1507b96589
Create Date: 2026-09-05 00:17:43.739171

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '6e848e2cf613'
down_revision = 'ab1507b96589'
branch_labels = None
depends_on = None


def upgrade():
    # listen_call_id records which live call a contact_case Insight AIcall is
    # currently listening to (docs/plans/
    # 2026-09-03-insight-ai-realtime-listen-design.md §5.8).
    #
    # It is a real column rather than a Metadata JSON key for exactly one
    # reason: EventCMCallHangup has to answer "which AIcalls are listening to
    # THIS call id?" and that is a WHERE clause, which JSON metadata cannot
    # serve. Hence the index.
    #
    # NOT NULL DEFAULT 0x00... (not NULL) matches how every other binary(16)
    # id column on this table stores "unset" -- Go's uuid.Nil round-trips
    # through it, and the Go struct field is a plain uuid.UUID, not a pointer.
    op.execute("""
        ALTER TABLE ai_aicalls
          ADD COLUMN listen_call_id BINARY(16) NOT NULL DEFAULT 0x00000000000000000000000000000000
    """)

    op.execute("""
        CREATE INDEX idx_ai_aicalls_listen_call_id ON ai_aicalls(listen_call_id)
    """)


def downgrade():
    op.execute("""
        DROP INDEX idx_ai_aicalls_listen_call_id ON ai_aicalls
    """)

    op.execute("""
        ALTER TABLE ai_aicalls
          DROP COLUMN listen_call_id
    """)
