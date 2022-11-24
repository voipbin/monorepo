"""add_column_calls_action_next_hold

Revision ID: d9dcf9645e24
Revises: c35b2644a075
Create Date: 2022-11-24 15:53:13.816176

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = 'd9dcf9645e24'
down_revision = 'c35b2644a075'
branch_labels = None
depends_on = None


def upgrade():
    op.execute("""alter table calls add column action_next_hold boolean default false after action;""")


def downgrade():
    op.execute("""alter table calls drop column action_next_hold;""")
