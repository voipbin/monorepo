"""Add index calls

Revision ID: e8b72d3bf93e
Revises: 0d7e0da11558
Create Date: 2022-04-25 01:12:31.511713

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'e8b72d3bf93e'
down_revision = '0d7e0da11558'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""create index idx_calls_customerid_and_tmcreate on calls(customer_id, tm_create);""")
    op.execute("""alter table calls add column flow_id binary(16) after bridge_id;""")


def downgrade():
    op.execute("""alter table calls drop index idx_calls_customerid_and_tmcreate;""")
    op.execute("""alter table calls drop column flow_id;""")
