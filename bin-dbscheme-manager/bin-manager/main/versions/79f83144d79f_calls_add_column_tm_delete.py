"""add column calls tm_delete

Revision ID: 79f83144d79f
Revises: fa7ab340d705
Create Date: 2023-01-01 00:02:23.189089

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '79f83144d79f'
down_revision = 'fa7ab340d705'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table calls add column tm_delete datetime(6) after tm_update;""")
    op.execute("""alter table channels add column tm_delete datetime(6) after tm_update;""")


def downgrade():
    op.execute("""alter table calls drop column tm_delete;""")
    op.execute("""alter table channels drop column tm_delete;""")
