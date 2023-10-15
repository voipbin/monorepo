"""routes add column name detail

Revision ID: dbbf8225587a
Revises: 6673c8ced2a2
Create Date: 2023-10-15 23:40:08.586269

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'dbbf8225587a'
down_revision = '6673c8ced2a2'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table routes add column name varchar(255) after customer_id;""")
    op.execute("""alter table routes add column detail text after name;""")


def downgrade():
    op.execute("""alter table routes drop column name;""")
    op.execute("""alter table routes drop column detail;""")

