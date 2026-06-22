"""agent_agents_drop_column_addresses

Revision ID: e234a24addec
Revises: ffeb3808129c
Create Date: 2026-06-22 00:00:01.000000

Drops the legacy agent_agents.addresses JSON column. This is the final
contract step in the agent address normalization sequence:

  1. Expand (14bca52528f5): agent_addresses child table created (non-unique index).
  2. Backfill: agent_addresses populated from JSON column (6 agents, 0 collisions).
  3. Contract step 1 (ffeb3808129c): UNIQUE(customer_id, type, target) promoted.
  4. Contract step 2 (this): legacy JSON column dropped.

The Go model (bin-agent-manager/models/agent/agent.go) already marks the
Addresses field as `db:"-"` (excluded from SELECT/INSERT). All reads and writes
go through the agent_addresses child table. The column is safe to drop.

downgrade() restores the column as a nullable JSON column so a rollback
is possible, but note that the previously stored JSON data will NOT be
restored — the column comes back empty. A data restore would require re-running
the backfill CLI in reverse, which is out of scope for this migration.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'e234a24addec'
down_revision = 'ffeb3808129c'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE agent_agents DROP COLUMN addresses;
    """)


def downgrade():
    # Restores the column structure only; data is NOT restored.
    op.execute("""
        ALTER TABLE agent_agents ADD COLUMN addresses json;
    """)
