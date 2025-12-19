"""add conferences actions

Revision ID: bebfd0decfa1
Revises: 6102c5d4793b
Create Date: 2021-11-05 00:55:01.190205

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'bebfd0decfa1'
down_revision = '6102c5d4793b'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table conferences add flow_id binary(16) after type;""")
    op.execute("""alter table conferences add confbridge_id binary(16) after flow_id;""")
    op.execute("""alter table conferences add pre_actions json after timeout;""")
    op.execute("""alter table conferences add post_actions json after pre_actions;""")

    op.execute("""create index idx_conferences_flow_id on conferences(flow_id);""")
    op.execute("""create index idx_conferences_confbridge_id on conferences(confbridge_id);""")


def downgrade():
    op.execute("""drop index idx_conferences_flow_id on conferences;""")
    op.execute("""drop index idx_conferences_confbridge_id on conferences;""")

    op.execute("""alter table conferences drop column flow_id;""")
    op.execute("""alter table conferences drop column confbridge_id;""")
    op.execute("""alter table conferences drop column pre_actions;""")
    op.execute("""alter table conferences drop column post_actions;""")
