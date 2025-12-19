"""remove webhook from recordings

Revision ID: 975995c9c2f8
Revises: 3d53561bfce6
Create Date: 2022-03-01 14:32:51.308912

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '975995c9c2f8'
down_revision = '3d53561bfce6'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table recordings drop column webhook_uri;""")


def downgrade():
    op.execute("""alter table recordings add webhook_uri varchar(1023);""")
