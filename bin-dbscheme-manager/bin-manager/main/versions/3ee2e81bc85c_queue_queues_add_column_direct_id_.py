"""queue_queues_add_column_direct_id_direct_hash

Revision ID: 3ee2e81bc85c
Revises: 6ab7d2a6b29c
Create Date: 2026-03-31 09:41:20.927829

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '3ee2e81bc85c'
down_revision = '6ab7d2a6b29c'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("ALTER TABLE queue_queues ADD COLUMN direct_id binary(16), ADD COLUMN direct_hash varchar(255)")


def downgrade():
    op.execute("ALTER TABLE queue_queues DROP COLUMN direct_id, DROP COLUMN direct_hash")
