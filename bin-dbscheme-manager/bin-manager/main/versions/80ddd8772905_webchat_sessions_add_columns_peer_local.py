"""webchat_sessions_add_columns_peer_local

Revision ID: 80ddd8772905
Revises: ffa2b1c5d1e6
Create Date: 2026-07-22 22:29:10.000000

Adds `peer`/`local` JSON columns to webchat_sessions. Per design doc
2026-07-22-webchat-session-referrer-peer-local-design.md §4.4's round-1
decision: nullable at the DB level (not NOT NULL) -- this is a short-lived,
high-churn table where the app layer already guarantees non-empty
commonaddress.Address values on every row created going forward, so a
BINARY(16)-to-UUID-string-style backfill for pre-existing rows is
disproportionate cost with no precedent elsewhere in this codebase's
migrations. Both `peer`/`local` are always present at the Go/JSON API layer
(no `omitempty` on Session.Peer/Session.Local) even though the DB column
itself is nullable.
"""
from alembic import op


# revision identifiers, used by Alembic.
revision = '80ddd8772905'
down_revision = 'ffa2b1c5d1e6'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE webchat_sessions
            ADD COLUMN peer  JSON NULL AFTER referrer,
            ADD COLUMN local JSON NULL AFTER peer;
    """)


def downgrade():
    op.execute("""
        ALTER TABLE webchat_sessions
            DROP COLUMN peer,
            DROP COLUMN local;
    """)
