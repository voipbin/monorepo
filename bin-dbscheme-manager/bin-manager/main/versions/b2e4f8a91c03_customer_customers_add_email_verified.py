"""customer_customers_add_email_verified

Revision ID: b2e4f8a91c03
Revises: 071504ef41d0
Create Date: 2026-02-08 12:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'b2e4f8a91c03'
down_revision = '071504ef41d0'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""ALTER TABLE customer_customers ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE AFTER billing_account_id;""")
    op.execute("""UPDATE customer_customers SET email_verified = TRUE WHERE tm_delete IS NULL;""")


def downgrade():
    op.execute("""ALTER TABLE customer_customers DROP COLUMN email_verified;""")
