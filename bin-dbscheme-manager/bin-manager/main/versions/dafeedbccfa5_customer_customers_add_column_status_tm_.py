"""customer_customers add column status tm_deletion_scheduled

Revision ID: dafeedbccfa5
Revises: b2c3d4e5f6a8
Create Date: 2026-02-17 01:39:32.659233

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = 'dafeedbccfa5'
down_revision = 'b2c3d4e5f6a8'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table customer_customers add column status varchar(16) not null default 'active' after email_verified;""")
    op.execute("""alter table customer_customers add column tm_deletion_scheduled datetime(6) default null after status;""")
    op.execute("""create index idx_customer_customers_status on customer_customers(status);""")
    op.execute("""update customer_customers set status = 'deleted' where tm_delete is not null;""")


def downgrade():
    op.execute("""alter table customer_customers drop index idx_customer_customers_status;""")
    op.execute("""alter table customer_customers drop column tm_deletion_scheduled;""")
    op.execute("""alter table customer_customers drop column status;""")
