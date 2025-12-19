"""call_recordings add column on_end_flow_id

Revision ID: 413135d8b469
Revises: c74277ac6ab8
Create Date: 2025-03-20 16:01:24.050981

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '413135d8b469'
down_revision = 'c74277ac6ab8'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table call_recordings add on_end_flow_id binary(16) after format;""")
    op.execute("""update call_recordings set on_end_flow_id = unhex(replace('00000000-0000-0000-0000-000000000000', '-', ''));""")
    op.execute("""create index idx_call_recordings_on_end_flow_id on call_recordings(on_end_flow_id);""")

def downgrade():
    op.execute("""alter table call_recordings drop column on_end_flow_id;""")
