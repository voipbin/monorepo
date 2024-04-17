"""add column dialroute_id and dialroutes

Revision ID: c35b2644a075
Revises: e1cb09d6f154
Create Date: 2022-11-06 01:02:30.573976

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'c35b2644a075'
down_revision = 'e1cb09d6f154'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table calls add column dialroute_id binary(16) default '' after hangup_reason;""")
    op.execute("""alter table calls add column dialroutes json after dialroute_id;""")


def downgrade():
    op.execute("""alter table calls drop column dialroute_id;""")
    op.execute("""alter table calls drop column dialroutes;""")
