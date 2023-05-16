"""add column groupcalls add linear

Revision ID: 71484d30e651
Revises: 34c7e8d98a8c
Create Date: 2023-04-24 17:29:52.633494

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '71484d30e651'
down_revision = '34c7e8d98a8c'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table groupcalls add column status varchar(16) after customer_id;""")
    op.execute("""alter table groupcalls add column flow_id binary(16) after status;""")
    op.execute("""alter table groupcalls add column master_call_id binary(16) after destinations;""")
    op.execute("""alter table groupcalls add column master_groupcall_id binary(16) after master_call_id;""")
    op.execute("""alter table groupcalls add column answer_groupcall_id binary(16) after call_ids;""")
    op.execute("""alter table groupcalls add column groupcall_ids json after answer_groupcall_id;""")
    op.execute("""alter table groupcalls add column groupcall_count integer after call_count;""")
    op.execute("""alter table groupcalls add column dial_index integer after call_count;""")

    op.execute("""update groupcalls set status = "";""")
    op.execute("""update groupcalls set flow_id = "";""")
    op.execute("""update groupcalls set master_call_id = "";""")
    op.execute("""update groupcalls set master_groupcall_id = "";""")
    op.execute("""update groupcalls set answer_groupcall_id = "";""")
    op.execute("""update groupcalls set groupcall_count = 0;""")
    op.execute("""update groupcalls set dial_index = 0;""")


def downgrade():
    op.execute("""alter table groupcalls drop column status;""")
    op.execute("""alter table groupcalls drop column flow_id;""")
    op.execute("""alter table groupcalls drop column master_call_id;""")
    op.execute("""alter table groupcalls drop column master_groupcall_id;""")
    op.execute("""alter table groupcalls drop column answer_groupcall_id;""")
    op.execute("""alter table groupcalls drop column groupcall_ids;""")
    op.execute("""alter table groupcalls drop column groupcall_count;""")
    op.execute("""alter table groupcalls drop column dial_index;""")

