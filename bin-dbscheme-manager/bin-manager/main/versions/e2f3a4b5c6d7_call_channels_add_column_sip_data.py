"""call_channels_add_column_sip_data

Revision ID: e2f3a4b5c6d7
Revises: d1a2b3c4d5e6
Create Date: 2026-03-08 12:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'e2f3a4b5c6d7'
down_revision = 'd1a2b3c4d5e6'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE call_channels ADD sip_data JSON DEFAULT NULL AFTER sip_transport;""")


def downgrade():
    op.execute("""ALTER TABLE call_channels DROP COLUMN sip_data;""")
