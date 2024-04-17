"""add column calls groupdial_id

Revision ID: 1849e8cf6d31
Revises: 88e160b5b6da
Create Date: 2023-03-01 02:41:30.710181

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '1849e8cf6d31'
down_revision = '88e160b5b6da'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table calls add column groupdial_id binary(16) after external_media_id;""")
    op.execute("""create index idx_calls_groupdial_id on calls(groupdial_id);""")

def downgrade():
    op.execute("""alter table calls drop column groupdial_id;""")

