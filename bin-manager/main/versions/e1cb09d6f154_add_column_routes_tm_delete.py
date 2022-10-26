"""add_column_routes_tm_delete

Revision ID: e1cb09d6f154
Revises: 32b74b818313
Create Date: 2022-10-26 18:52:55.553526

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'e1cb09d6f154'
down_revision = '32b74b818313'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table routes add column tm_delete datetime(6) after tm_update;""")


def downgrade():
    op.execute("""alter table routes drop column tm_delete;""")
