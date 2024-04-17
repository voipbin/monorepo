"""Remove_queue_duration

Revision ID: 476c036bd925
Revises: e93a0bfa95fe
Create Date: 2022-05-13 16:50:02.795194

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '476c036bd925'
down_revision = 'e93a0bfa95fe'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table queues drop column total_wait_duration;""")
    op.execute("""alter table queues drop column total_service_duration;""")

    op.execute("""alter table queuecalls add duration_waiting integer default 0 after timeout_service;""")
    op.execute("""alter table queuecalls add duration_service integer default 0 after duration_waiting;""")

def downgrade():
    op.execute("""alter table queues add total_wait_duration integer default 0 after total_abandoned_count;""")
    op.execute("""alter table queues add total_service_duration integer default 0 after total_wait_duration;""")

    op.execute("""alter table queuecalls drop column duration_waiting;""")
    op.execute("""alter table queuecalls drop column duration_service;""")
