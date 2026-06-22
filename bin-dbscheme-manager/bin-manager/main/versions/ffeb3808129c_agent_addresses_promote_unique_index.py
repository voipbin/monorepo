"""agent_addresses_promote_unique_index

Revision ID: ffeb3808129c
Revises: 14bca52528f5
Create Date: 2026-06-22 00:00:00.000000

Promotes the non-unique lookup index on agent_addresses(customer_id, type, target)
to a UNIQUE constraint. This is the contract step that follows:

  1. Expand migration (14bca52528f5): created the table with a non-unique index.
  2. Backfill: populated agent_addresses from agent_agents.addresses JSON column
     (6 agents, 8 rows, 0 collisions — verified 2026-06-22).
  3. Duplicate-resolution gate: 0 duplicates confirmed before this migration.

After this migration, attempting to assign the same (customer_id, type, target)
to more than one agent row will be rejected at the DB layer with errno 1062,
which the agenthandler maps to a 409 AlreadyExists error.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ffeb3808129c'
down_revision = '14bca52528f5'
branch_labels = None
depends_on = None


def upgrade():
    # Drop the non-unique lookup index first, then recreate as UNIQUE.
    op.execute("""
        DROP INDEX idx_agent_addresses_owner ON agent_addresses;
    """)
    op.execute("""
        CREATE UNIQUE INDEX idx_agent_addresses_owner
            ON agent_addresses(customer_id, type, target);
    """)


def downgrade():
    # Revert to a non-unique index (expand state).
    op.execute("""
        DROP INDEX idx_agent_addresses_owner ON agent_addresses;
    """)
    op.execute("""
        CREATE INDEX idx_agent_addresses_owner
            ON agent_addresses(customer_id, type, target);
    """)
