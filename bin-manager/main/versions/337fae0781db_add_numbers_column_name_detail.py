"""add numbers column name detail

Revision ID: 337fae0781db
Revises: 5c2ed766c8ff
Create Date: 2022-02-07 11:55:45.971944

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '337fae0781db'
down_revision = '5c2ed766c8ff'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table numbers add column name varchar(255) default '' after customer_id;""")
    op.execute("""alter table numbers add column detail text after name;""")


def downgrade():
    op.execute("""alter table numbers drop column name;""")
    op.execute("""alter table numbers drop column detail;""")
