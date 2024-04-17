"""add column groupcalls call_count

Revision ID: 0861a73c53c7
Revises: 328dce97f0d8
Create Date: 2023-04-12 03:50:43.428226

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '0861a73c53c7'
down_revision = '328dce97f0d8'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table groupcalls add call_count integer after call_ids;""")
    op.execute("""update groupcalls set call_count = 0;""")


def downgrade():
    op.execute("""alter table groupcalls drop column call_count;""")
