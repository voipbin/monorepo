"""billing_accounts add column plan_status

Revision ID: 6ab7d2a6b29c
Revises: 08d686855c69
Create Date: 2026-03-29 01:31:53.829911

"""
from alembic import op


# revision identifiers, used by Alembic.
revision = '6ab7d2a6b29c'
down_revision = '08d686855c69'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table billing_accounts add column plan_status varchar(32) not null default 'active' after plan_type;""")


def downgrade():
    op.execute("""alter table billing_accounts drop column plan_status;""")
