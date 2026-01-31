"""ai_ais add column tool_names

Revision ID: 1ef85e7392ce
Revises: 5b7401675bcc
Create Date: 2026-01-31 23:07:44.205188

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '1ef85e7392ce'
down_revision = '5b7401675bcc'
branch_labels = None
depends_on = None


def upgrade():
    # Add tool_names column (json array of enabled tool names for this AI)
    # Default NULL means no tools enabled, ["all"] enables all tools
    op.execute("""alter table ai_ais add column tool_names json after stt_type;""")

    # Set all existing AIs to have all tools enabled for backwards compatibility
    op.execute("""update ai_ais set tool_names = '["all"]' where tool_names is null;""")


def downgrade():
    op.execute("""alter table ai_ais drop column tool_names;""")
