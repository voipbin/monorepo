"""contact_interactions_add_local_type_local_target

Revision ID: adb8daac2bb0
Revises: bbcf80d332eb
Create Date: 2026-06-28

Adds local_type and local_target to contact_interactions (VOIP-1208).

The original table (VOIP-1206, ac5d4e18060c) stored only the remote party
(peer_type / peer_target) because read-time identity resolution only needs the
peer. However, the local endpoint (our number / LINE official account / email)
is also a first-class immutable fact carried by the event at projection time.
Omitting it under the no-backfill policy would cause permanent data loss for
any future per-number/per-channel inbound attribution use case.

Decision A (VOIP-1208 scope): scoped here because the projection handler fills
these values; keeping schema change and handler in the same work unit.

Both columns:
- NOT NULL DEFAULT '' (consistent with peer_type / peer_target convention).
- NOT in the idempotency unique (same event always has same local; widening
  the unique would add no dedup benefit).
- No index (not used as match key or cursor; attribution only).
- downgrade drops both columns.
"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'adb8daac2bb0'
down_revision = 'bbcf80d332eb'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE contact_interactions
            ADD COLUMN local_type   VARCHAR(255) NOT NULL DEFAULT '' AFTER peer_target,
            ADD COLUMN local_target VARCHAR(255) NOT NULL DEFAULT '' AFTER local_type;
    """)


def downgrade():
    op.execute("""
        ALTER TABLE contact_interactions
            DROP COLUMN local_target,
            DROP COLUMN local_type;
    """)
