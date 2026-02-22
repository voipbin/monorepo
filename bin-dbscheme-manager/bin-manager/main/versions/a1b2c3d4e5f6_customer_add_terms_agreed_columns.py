"""customer_add_terms_agreed_columns

Revision ID: a1b2c3d4e5f6
Revises: 8f42ac0555e7
Create Date: 2026-02-22 00:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'a1b2c3d4e5f6'
down_revision = '8f42ac0555e7'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE customer_customers
            ADD COLUMN terms_agreed_version VARCHAR(64) NOT NULL DEFAULT '' AFTER status,
            ADD COLUMN terms_agreed_ip VARCHAR(45) NOT NULL DEFAULT '' AFTER terms_agreed_version;
    """)


def downgrade():
    op.execute("""
        ALTER TABLE customer_customers
            DROP COLUMN terms_agreed_ip,
            DROP COLUMN terms_agreed_version;
    """)
