"""add users name detail

Revision ID: af6321e8bdef
Revises: 7e1b676ced05
Create Date: 2021-11-21 15:54:54.454776

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'af6321e8bdef'
down_revision = '7e1b676ced05'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table users add column name varchar(255) after password_hash;""")
    op.execute("""alter table users add column detail text after name;""")

def downgrade():
    op.execute("""alter table users drop column name;""")
    op.execute("""alter table users drop column detail;""")
