"""queue_queuecalls remove column exit_action_id

Revision ID: 982e06d7691e
Revises: bab94f08a5be
Create Date: 2025-03-10 21:28:52.997838

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '982e06d7691e'
down_revision = 'bab94f08a5be'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table queue_queuecalls drop column exit_action_id;""")


def downgrade():
    op.execute("""alter table queue_queuecalls add exit_action_id binary(16) after forward_action_id;""")
    op.execute("""UPDATE queue_queuecalls SET exit_action_id = unhex(replace('00000000-0000-0000-0000-000000000000', '-', ''));""")

    
    
