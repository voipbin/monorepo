"""webchat_sessions_add_column_referrer

Revision ID: ffa2b1c5d1e6
Revises: 04b99363284c
Create Date: 2026-07-22 22:28:30.340236

Adds `referrer` to webchat_sessions: document.referrer at session-creation
time, captured client-side. Mirrors the existing `page_url` column (added in
04b99363284c) -- same nullable VARCHAR(2048) shape, same "best-effort,
captured once at Create() time" semantics. See docs/plans/
2026-07-22-webchat-session-referrer-peer-local-design.md.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ffa2b1c5d1e6'
down_revision = '04b99363284c'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE webchat_sessions ADD COLUMN referrer VARCHAR(2048) NULL;""")


def downgrade():
    op.execute("""ALTER TABLE webchat_sessions DROP COLUMN referrer;""")
