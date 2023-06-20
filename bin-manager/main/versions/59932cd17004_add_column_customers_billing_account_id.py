"""add column customers billing account id

Revision ID: 59932cd17004
Revises: d035a214f7da
Create Date: 2023-06-19 18:39:22.024710

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '59932cd17004'
down_revision = 'd035a214f7da'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table customers add billing_account_id binary(16) after permission_ids;""")
    op.execute("""update customers set billing_account_id = "";""")


def downgrade():
    op.execute("""alter table customers drop column billing_account_id;""")
