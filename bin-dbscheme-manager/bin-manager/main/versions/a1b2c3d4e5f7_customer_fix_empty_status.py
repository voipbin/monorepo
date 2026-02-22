"""customer_customers fix empty status values

Revision ID: a1b2c3d4e5f7
Revises: 455debd049b2
Create Date: 2026-02-23 00:00:00.000000

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'a1b2c3d4e5f7'
down_revision = '455debd049b2'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""update customer_customers set status = 'active' where status = '' and email_verified = 1;""")
    op.execute("""update customer_customers set status = 'initial' where status = '' and email_verified = 0;""")


def downgrade():
    op.execute("""update customer_customers set status = '' where status = 'initial';""")
    op.execute("""update customer_customers set status = '' where status = 'active' and tm_delete is null;""")
