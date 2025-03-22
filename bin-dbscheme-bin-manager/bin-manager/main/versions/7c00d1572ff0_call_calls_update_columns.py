"""call_calls update columns

Revision ID: 7c00d1572ff0
Revises: 413135d8b469
Create Date: 2025-03-22 22:14:13.853735

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '7c00d1572ff0'
down_revision = '413135d8b469'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table call_calls change column active_flow_id activeflow_id binary(16);""")
    op.execute("""create index idx_call_calls_activeflow_id on call_calls(activeflow_id);""")

    op.execute("""alter table call_confbridges add column activeflow_id binary(16) after customer_id;""")
    op.execute("""update call_confbridges set activeflow_id = unhex(replace('00000000-0000-0000-0000-000000000000', '-', ''));""")
    op.execute("""create index idx_call_confbridges_activeflow_id on call_confbridges(activeflow_id);""")

    op.execute("""alter table call_confbridges add column reference_type varchar(255) after activeflow_id;""")
    op.execute("""update call_confbridges set reference_type = '';""")

    op.execute("""alter table call_confbridges add column reference_id binary(16) after reference_type;""")
    op.execute("""update call_confbridges set reference_id = unhex(replace('00000000-0000-0000-0000-000000000000', '-', ''));""")
    op.execute("""create index idx_call_confbridges_reference_id on call_confbridges(reference_id);""")

    op.execute("""alter table call_recordings add column activeflow_id binary(16) after owner_id;""")
    op.execute("""update call_recordings set activeflow_id = unhex(replace('00000000-0000-0000-0000-000000000000', '-', ''));""")
    op.execute("""create index idx_call_recordings_activeflow_id on call_recordings(activeflow_id);""")


def downgrade():
    op.execute("""alter table call_calls change column activeflow_id active_flow_id binary(16);""")

    op.execute("""alter table call_confbridges drop column activeflow_id;""")
    op.execute("""alter table call_confbridges drop column reference_type;""")
    op.execute("""alter table call_confbridges drop column reference_id;""")

    op.execute("""alter table call_recordings drop column activeflow_id;""")
