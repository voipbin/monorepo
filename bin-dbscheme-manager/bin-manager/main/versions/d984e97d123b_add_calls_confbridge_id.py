"""add calls confbridge_id

Revision ID: d984e97d123b
Revises: ae08184032e6
Create Date: 2021-12-13 07:51:19.006503

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'd984e97d123b'
down_revision = 'ae08184032e6'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table calls add confbridge_id binary(16) after flow_id;""")
    op.execute("""alter table calls drop column conference_id;""")

    op.execute("""drop index idx_confbridges_conference_id on confbridges;""")
    op.execute("""alter table confbridges drop column conference_id;""")


def downgrade():
    op.execute("""alter table calls add conference_id binary(16) after flow_id;""")
    op.execute("""alter table calls drop column confbridge_id;""")

    op.execute("""create index idx_confbridges_conference_id on confbridges(conference_id);""")
    op.execute("""alter table confbridges add conference_id binary(16) after id;""")

