"""storage_files add column filename

Revision ID: 68a9b4bd0e5c
Revises: 58557426bd74
Create Date: 2024-05-21 15:24:26.641340

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '68a9b4bd0e5c'
down_revision = '58557426bd74'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table storage_files add column filename varchar(1023);""")
    op.execute("""update storage_files set filename = '';""")


def downgrade():
    op.execute("""alter table storage_files drop column filename;""")
