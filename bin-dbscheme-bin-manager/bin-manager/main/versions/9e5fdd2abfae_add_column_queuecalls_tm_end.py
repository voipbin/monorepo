"""add column queuecalls tm end

Revision ID: 9e5fdd2abfae
Revises: 9c150d002975
Create Date: 2023-02-15 14:57:05.624604

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '9e5fdd2abfae'
down_revision = '9c150d002975'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table queuecalls add column tm_end datetime(6) after tm_update;""")
    op.execute("""alter table queuecalls change conference_id confbridge_id binary(16);""")



def downgrade():
    op.execute("""alter table queuecalls drop column tm_end;""")
    op.execute("""alter table queuecalls change confbridge_id conference_id binary(16);""")
