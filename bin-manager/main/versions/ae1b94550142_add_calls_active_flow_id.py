"""add calls active_flow_id

Revision ID: ae1b94550142
Revises: 60ad39e69170
Create Date: 2022-03-20 05:20:56.507200

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ae1b94550142'
down_revision = '60ad39e69170'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table calls add column active_flow_id binary(16) after flow_id;""")


def downgrade():
    op.execute("""alter table calls drop column active_flow_id;""")
