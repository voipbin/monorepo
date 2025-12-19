"""add_flows_type

Revision ID: 016e981efa60
Revises: 6b7f1dea94de
Create Date: 2022-01-03 05:54:37.309031

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '016e981efa60'
down_revision = '6b7f1dea94de'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table flows add type varchar(255) after user_id;""")


def downgrade():
    op.execute("""alter table flows drop column type;""")
