"""add numbers callflowid messageflowid

Revision ID: 45acd7839db4
Revises: ae1b94550142
Create Date: 2022-03-21 01:40:42.485051

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '45acd7839db4'
down_revision = 'ae1b94550142'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table numbers drop column flow_id;""")
    op.execute("""alter table numbers add column call_flow_id binary(16) after customer_id;""")
    op.execute("""alter table numbers add column message_flow_id binary(16) after call_flow_id;""")
    op.execute("""create index idx_numbers_call_flow_id on numbers(call_flow_id);""")
    op.execute("""create index idx_numbers_message_flow_id on numbers(message_flow_id);""")


def downgrade():
    op.execute("""alter table numbers add column flow_id binary(16) after customer_id;""")
    op.execute("""create index idx_numbers_flow_id on numbers(flow_id);""")
    op.execute("""alter table numbers drop column call_flow_id;""")
    op.execute("""alter table numbers drop column message_flow_id;""")


