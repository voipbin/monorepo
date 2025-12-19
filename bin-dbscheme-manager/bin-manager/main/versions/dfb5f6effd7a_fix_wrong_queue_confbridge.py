"""fix wrong queue confbridge

Revision ID: dfb5f6effd7a
Revises: 016e981efa60
Create Date: 2022-01-16 13:31:26.991923

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'dfb5f6effd7a'
down_revision = '016e981efa60'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table queues drop column flow_id;""")
    op.execute("""alter table queues drop column confbridge_id;""")
    op.execute("""alter table queues drop column forward_action_id;""")

    op.execute("""alter table queuecalls add column flow_id binary(16) after reference_id;""")


def downgrade():
    op.execute("""alter table queues add column flow_id binary(16) after user_id;""")
    op.execute("""alter table queues add column confbridge_id binary(16) after flow_id;""")
    op.execute("""alter table queues add column forward_action_id binary(16) after confbridge_id;""")

    op.execute("""alter table queuecalls drop column flow_id;""")
