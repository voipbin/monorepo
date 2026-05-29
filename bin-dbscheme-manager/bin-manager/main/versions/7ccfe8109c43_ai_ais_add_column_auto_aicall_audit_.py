"""ai_ais_add_column_auto_aicall_audit_enabled

Revision ID: 7ccfe8109c43
Revises: 43a66dafdec9
Create Date: 2026-05-30 02:20:07.333597

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '7ccfe8109c43'
down_revision = '43a66dafdec9'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE ai_ais ADD auto_aicall_audit_enabled TINYINT(1) NOT NULL DEFAULT 0 AFTER smart_turn_enabled;""")


def downgrade():
    op.execute("""ALTER TABLE ai_ais DROP COLUMN auto_aicall_audit_enabled;""")
