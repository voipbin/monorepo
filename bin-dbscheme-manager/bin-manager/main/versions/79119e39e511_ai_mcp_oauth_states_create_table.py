"""ai_mcp_oauth_states_create_table

Revision ID: 79119e39e511
Revises: 62c10f986f07
Create Date: 2026-09-12 06:10:24.616473

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '79119e39e511'
down_revision = '62c10f986f07'
branch_labels = None
depends_on = None


def upgrade():
    # Short-lived CSRF/PKCE state for in-flight OAuth authorization
    # requests, see
    # docs/plans/2026-09-12-mcp-server-oauth-support-design.md §5. NOT
    # part of ai_mcp_servers: a flow is initiated before the target row
    # necessarily exists, and this data lives seconds, not the lifetime
    # of the entity. No FKs enforced at the DB layer, matching this
    # monorepo's existing app-layer referential integrity convention for
    # ai_* tables.
    op.execute("""
        create table ai_mcp_oauth_states(
          state         varchar(64)  not null,
          customer_id   binary(16)   not null,
          mcp_server_id binary(16),
          vendor        varchar(64)  not null,
          pkce_verifier varchar(128) not null,

          tm_create datetime(6) not null,
          tm_expire datetime(6) not null,

          primary key(state)
        );
    """)

    op.execute("""create index idx_ai_mcp_oauth_states_customer_id on ai_mcp_oauth_states(customer_id);""")
    op.execute("""create index idx_ai_mcp_oauth_states_tm_expire on ai_mcp_oauth_states(tm_expire);""")


def downgrade():
    op.execute("""drop table ai_mcp_oauth_states;""")
