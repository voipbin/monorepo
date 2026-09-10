"""ai_mcp_servers_create_table

Revision ID: 9b0ad37e0360
Revises: 42cf2ffc4273
Create Date: 2026-09-11 05:42:59.375008

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '9b0ad37e0360'
down_revision = '42cf2ffc4273'
branch_labels = None
depends_on = None


def upgrade():
    # Customer-registered remote MCP (Model Context Protocol) server, see
    # docs/plans/2026-09-11-mcp-tool-integration-design.md §5.
    op.execute("""
        create table ai_mcp_servers(
          id                binary(16),
          customer_id       binary(16),

          name              varchar(255),
          detail            text,

          url               varchar(2048),
          status            varchar(16),

          auth_type         varchar(16),
          api_key_header    varchar(255),
          secret_ciphertext blob,
          secret_nonce      binary(12),
          key_version       smallint,

          tm_create datetime(6),
          tm_update datetime(6),
          tm_delete datetime(6),

          primary key(id)
        );
    """)

    op.execute("""create index idx_ai_mcp_servers_create on ai_mcp_servers(tm_create);""")
    op.execute("""create index idx_ai_mcp_servers_customer_id on ai_mcp_servers(customer_id);""")


def downgrade():
    op.execute("""drop table ai_mcp_servers;""")
