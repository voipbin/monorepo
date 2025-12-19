"""remove webhook from calls conferences

Revision ID: 1a8a38f9c167
Revises: 431ea0ae1aab
Create Date: 2022-02-03 15:02:49.668186

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '1a8a38f9c167'
down_revision = '431ea0ae1aab'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table calls drop column webhook_uri;""")
    op.execute("""alter table flows drop column webhook_uri;""")
    op.execute("""alter table conferences drop column webhook_uri;""")


def downgrade():
    op.execute("""alter table calls add webhook_uri varchar(1023);""")
    op.execute("""alter table flows add webhook_uri after varchar(1023);""")
    op.execute("""alter table conferences add webhook_uri varchar(1023);""")
