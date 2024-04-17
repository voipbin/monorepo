"""remove conferences bridge_id

Revision ID: 7e1b676ced05
Revises: bebfd0decfa1
Create Date: 2021-11-12 00:44:59.719485

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '7e1b676ced05'
down_revision = 'bebfd0decfa1'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table conferences drop column bridge_id;""")


def downgrade():
    op.execute("""alter table conferences add bridge_id varchar(255) after type;""")

