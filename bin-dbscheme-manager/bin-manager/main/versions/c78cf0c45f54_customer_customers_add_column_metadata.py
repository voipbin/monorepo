"""customer_customers add column metadata

Revision ID: c78cf0c45f54
Revises: f2b3c4d5e6f7
Create Date: 2026-03-05 10:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'c78cf0c45f54'
down_revision = 'f2b3c4d5e6f7'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE customer_customers ADD COLUMN metadata JSON DEFAULT NULL;""")


def downgrade():
    op.execute("""ALTER TABLE customer_customers DROP COLUMN metadata;""")
