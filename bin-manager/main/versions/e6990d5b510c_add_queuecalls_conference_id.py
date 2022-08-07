"""add_queuecalls_conference_id

Revision ID: e6990d5b510c
Revises: 1ea46dfdb2dc
Create Date: 2022-08-08 01:42:06.043541

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'e6990d5b510c'
down_revision = '1ea46dfdb2dc'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table queuecalls drop column confbridge_id;""")

    op.execute("""alter table queuecalls add column conference_id binary(16) default '' after exit_action_id;""")



def downgrade():
    op.execute("""alter table queuecalls drop column conference_id;""")

    op.execute("""alter table queuecalls add column confbridge_id binary(16) default '' after exit_action_id;""")

