"""ai_mcp_servers_add_oauth_columns

Revision ID: 62c10f986f07
Revises: d8e342656cf0
Create Date: 2026-09-12 06:10:24.337690

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '62c10f986f07'
down_revision = 'd8e342656cf0'
branch_labels = None
depends_on = None


def upgrade():
    # New oauth auth_type support, see
    # docs/plans/2026-09-12-mcp-server-oauth-support-design.md §5. All
    # columns nullable, no backfill -- every existing row is
    # auth_type IN ('', 'bearer', 'api_key') and none of these apply.
    # Types match the EXACT existing secret_ciphertext/secret_nonce
    # columns in this table (blob / binary(12) -- AES-GCM nonce is
    # always exactly 12 bytes), not a different convention.
    op.execute("""
        alter table ai_mcp_servers
          add column oauth_vendor             varchar(64)  null,
          add column access_token_ciphertext  blob         null,
          add column access_token_nonce       binary(12)   null,
          add column access_token_expires_at  datetime(6)  null,
          add column refresh_token_ciphertext blob         null,
          add column refresh_token_nonce      binary(12)   null;
    """)


def downgrade():
    op.execute("""
        alter table ai_mcp_servers
          drop column oauth_vendor,
          drop column access_token_ciphertext,
          drop column access_token_nonce,
          drop column access_token_expires_at,
          drop column refresh_token_ciphertext,
          drop column refresh_token_nonce;
    """)
