"""agent_addresses_create_table

Revision ID: 14bca52528f5
Revises: e799944f33bc
Create Date: 2026-06-21 23:30:42.957410

Splits agent endpoint addresses out of the agent_agents.addresses JSON column
into a normalized child table. See
docs/plans/2026-06-21-add-agent-addresses-table-design.md.

This migration creates the table with a NON-unique lookup index only. The
UNIQUE(customer_id, type, target) constraint is promoted in a separate
operational step AFTER the data backfill and a duplicate-resolution gate, so the
backfill can never abort mid-way on a pre-existing duplicate (design HIGH#1).
The agent_agents.addresses JSON column is intentionally left in place; it is
dropped in a later follow-up migration once the table split is verified in prod.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '14bca52528f5'
down_revision = 'e799944f33bc'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        create table agent_addresses(
            -- identity
            id          binary(16),   -- surrogate id
            agent_id    binary(16),   -- owning agent (agent_agents.id)
            customer_id binary(16),   -- denormalized for the by-(customer,addr) lookup + uniqueness

            -- address (commonaddress.Address)
            type        varchar(255),
            target      varchar(255),
            target_name varchar(255),
            name        varchar(255),
            detail      text,

            idx         int,          -- preserves address order within an agent (ring order)

            tm_create   datetime(6),
            tm_update   datetime(6),

            primary key(id)
        );
    """)

    op.execute("""
        create index idx_agent_addresses_agent_id on agent_addresses(agent_id);
    """)
    # non-unique lookup index (hot-path by-address owner lookup). Promoted to a
    # UNIQUE index in a separate step after the backfill + duplicate gate.
    op.execute("""
        create index idx_agent_addresses_owner on agent_addresses(customer_id, type, target);
    """)


def downgrade():
    op.execute("""
        drop table agent_addresses;
    """)
