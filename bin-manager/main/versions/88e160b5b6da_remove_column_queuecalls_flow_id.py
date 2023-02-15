"""remove column queuecalls flow_id

Revision ID: 88e160b5b6da
Revises: 9e5fdd2abfae
Create Date: 2023-02-16 02:50:36.697726

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '88e160b5b6da'
down_revision = '9e5fdd2abfae'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table queuecalls drop column flow_id;""")


def downgrade():
    op.execute("""alter table queuecalls add column flow_id binary(16) after confbridge_id;""")

