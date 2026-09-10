"""ai_ais_add_column_mcp_server_ids

Revision ID: d8e342656cf0
Revises: 9b0ad37e0360
Create Date: 2026-09-11 05:43:00.969035

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'd8e342656cf0'
down_revision = '9b0ad37e0360'
branch_labels = None
depends_on = None


def upgrade():
    # Customer-registered McpServer id whitelist for this AI, see
    # docs/plans/2026-09-11-mcp-tool-integration-design.md §5.
    op.execute("""alter table ai_ais add column mcp_server_ids json after tool_names;""")


def downgrade():
    op.execute("""alter table ai_ais drop column mcp_server_ids;""")
