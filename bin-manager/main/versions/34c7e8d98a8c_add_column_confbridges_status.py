"""add column confbridges status

Revision ID: 34c7e8d98a8c
Revises: 0861a73c53c7
Create Date: 2023-04-18 14:53:03.067735

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '34c7e8d98a8c'
down_revision = '0861a73c53c7'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table confbridges add status varchar(16) after type;""")
    op.execute("""update confbridges set status = '';""")


def downgrade():
    op.execute("""alter table confbridges drop column status;""")
