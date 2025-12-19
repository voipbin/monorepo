"""queue_queues ADD COLUMN wait_flow_id

Revision ID: 04877993e57e
Revises: 268e0c57e1ec
Create Date: 2025-04-24 23:15:23.844156

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '04877993e57e'
down_revision = '268e0c57e1ec'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""
        ALTER TABLE queue_queues 
        ADD COLUMN wait_flow_id binary(16) 
        AFTER wait_actions;
    """)
    op.execute("""
        UPDATE queue_queues 
        SET wait_flow_id = UNHEX(REPLACE('00000000-0000-0000-0000-000000000000', '-', ''));
    """)
    op.execute("""
        ALTER TABLE queue_queues 
        DROP COLUMN wait_actions;
    """)


def downgrade():
    op.execute("""
        ALTER TABLE queue_queues 
        ADD COLUMN wait_actions JSON 
        AFTER wait_flow_id;
    """)
    op.execute("""
        UPDATE queue_queues 
        SET wait_actions = '{}';
    """)
    op.execute("""
        ALTER TABLE queue_queues 
        DROP COLUMN wait_flow_id;
    """)
