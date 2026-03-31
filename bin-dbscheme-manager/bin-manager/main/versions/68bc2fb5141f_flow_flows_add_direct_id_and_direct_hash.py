"""flow_flows_add_direct_id_and_direct_hash

Revision ID: 68bc2fb5141f
Revises: 3ee2e81bc85c
Create Date: 2026-03-31 15:31:16.599274

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '68bc2fb5141f'
down_revision = '3ee2e81bc85c'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("ALTER TABLE flow_flows ADD COLUMN direct_id binary(16), ADD COLUMN direct_hash varchar(255)")


def downgrade():
    op.execute("ALTER TABLE flow_flows DROP COLUMN direct_id, DROP COLUMN direct_hash")
