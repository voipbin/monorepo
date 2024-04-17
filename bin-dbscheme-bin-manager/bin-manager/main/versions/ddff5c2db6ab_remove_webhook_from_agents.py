"""remove webhook from agents

Revision ID: ddff5c2db6ab
Revises: 337fae0781db
Create Date: 2022-02-09 09:58:47.984576

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'ddff5c2db6ab'
down_revision = '337fae0781db'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table agents drop column webhook_method;""")
    op.execute("""alter table agents drop column webhook_uri;""")


def downgrade():
    op.execute("""alter table agents add webhook_method varchar(255);""")
    op.execute("""alter table agents add webhook_uri varchar(1023);""")
