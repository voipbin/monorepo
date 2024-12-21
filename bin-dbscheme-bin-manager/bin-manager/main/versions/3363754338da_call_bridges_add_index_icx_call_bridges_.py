"""call_bridges add index icx_call_bridges_id

Revision ID: 3363754338da
Revises: 133ea3a35091
Create Date: 2024-12-21 15:48:01.520475

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '3363754338da'
down_revision = '133ea3a35091'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""create index idx_call_bridges_id on call_bridges(id);""")
    op.execute("""create index idx_call_bridges_asterisk_id_id on call_bridges(asterisk_id, id);""")


def downgrade():
    op.execute("""alter table call_bridges drop index idx_call_bridges_id;""")
    op.execute("""alter table call_bridges drop index idx_call_bridges_asterisk_id_id;""")
