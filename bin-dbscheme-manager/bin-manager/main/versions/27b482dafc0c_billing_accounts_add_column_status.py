"""billing_accounts add column status

Revision ID: 27b482dafc0c
Revises: dafeedbccfa5
Create Date: 2026-02-17 01:39:37.090341

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = '27b482dafc0c'
down_revision = 'dafeedbccfa5'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table billing_accounts add column status varchar(16) not null default 'active' after customer_id;""")
    op.execute("""create index idx_billing_accounts_status on billing_accounts(status);""")
    op.execute("""update billing_accounts set status = 'deleted' where tm_delete is not null;""")


def downgrade():
    op.execute("""alter table billing_accounts drop index idx_billing_accounts_status;""")
    op.execute("""alter table billing_accounts drop column status;""")
