"""add join bridge

Revision ID: 9443d2d65ad8
Revises: 61d3d8827efd
Create Date: 2021-09-13 13:16:26.603366

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '9443d2d65ad8'
down_revision = '61d3d8827efd'
branch_labels = None
depends_on = None


def upgrade():
    try:
        #bridges
        op.execute("""alter table bridges drop column conference_id;""")
        op.execute("""alter table bridges drop column conference_type;""")
        op.execute("""alter table bridges drop column conference_join;""")
        op.execute("""alter table bridges add column reference_type varchar(255) after channel_ids;""")
        op.execute("""alter table bridges add column reference_id binary(16) after reference_type;""")
        op.execute("""create index idx_bridges_reference_id on bridges(reference_id);""")

        #calls
        op.execute("""alter table calls add column bridge_id varchar(255) after channel_id;;""")
    except:
        pass


def downgrade():
    # nothing to do.
    pass
