"""remove webhook from queue

Revision ID: 3d53561bfce6
Revises: ddff5c2db6ab
Create Date: 2022-02-10 01:09:50.604444

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '3d53561bfce6'
down_revision = 'ddff5c2db6ab'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table queuecalls drop column webhook_uri;""")
    op.execute("""alter table queuecalls drop column webhook_method;""")

    op.execute("""alter table queues drop column webhook_uri;""")
    op.execute("""alter table queues drop column webhook_method;""")


def downgrade():
    op.execute("""alter table queuecalls add webhook_uri varchar(1023) after confbridge_id;""")
    op.execute("""alter table queuecalls add webhook_method varchar(255) after webhook_uri;""")

    op.execute("""alter table queues add webhook_uri varchar(1023) after detail;""")
    op.execute("""alter table queues add webhook_method varchar(255) after webhook_uri;""")
