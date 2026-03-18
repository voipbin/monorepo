"""customer_customers add identity_verification_status column

Revision ID: ffb1bfe3f8d7
Revises: 889f8fc50634
Create Date: 2026-03-18 12:00:00.000000

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ffb1bfe3f8d7'
down_revision = '889f8fc50634'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE customer_customers ADD COLUMN identity_verification_status VARCHAR(32) NOT NULL DEFAULT 'none';""")
    # Grandfather all existing active customers to 'verified' so they are not disrupted.
    # New customers created after this migration will default to 'none'.
    op.execute("""UPDATE customer_customers SET identity_verification_status = 'verified' WHERE status = 'active';""")


def downgrade():
    op.execute("""ALTER TABLE customer_customers DROP COLUMN identity_verification_status;""")
