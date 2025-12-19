"""rename_column_calls_groupdial

Revision ID: aa8826443972
Revises: 1849e8cf6d31
Create Date: 2023-03-05 16:54:13.725145

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'aa8826443972'
down_revision = '1849e8cf6d31'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table calls change groupdial_id groupcall_id binary(16);""")


def downgrade():
    op.execute("""alter table calls change groupcall_id groupdial_id binary(16);""")
