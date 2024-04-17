"""add column conferencecalls tm_end

Revision ID: c66d262cc66c
Revises: a2fc7a664d4f
Create Date: 2023-01-15 23:10:54.863016

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'c66d262cc66c'
down_revision = 'a2fc7a664d4f'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table conferences add tm_end datetime(6) after recording_ids;""")


def downgrade():
    op.execute("""alter table conferences drop column tm_end;""")
